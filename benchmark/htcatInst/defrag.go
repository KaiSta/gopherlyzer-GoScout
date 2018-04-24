package htcat

import (
	"io"
	"sync/atomic"

	"../tracer"
)

type writerToCloser interface {
	io.WriterTo
	io.Closer
}

type fragment struct {
	ord      int64
	contents writerToCloser
}

type defrag struct {
	lastWritten int64

	lastAlloc int64

	lastOrdinal       int64
	lastOrdinalNotify chan struct {
		threadId uint64
		value    int64
	}

	future map[int64]*fragment

	registerNotify chan struct {
		threadId uint64
		value    *fragment
	}

	cancellation error
	cancelNotify chan struct {
		threadId uint64
		value    error
	}

	written int64

	done chan struct {
		threadId uint64
		value    struct{}
	}
}

func newDefrag() *defrag {
	ret := defrag{}
	ret.initDefrag()

	return &ret
}

func (d *defrag) initDefrag() {
	d.future = make(map[int64]*fragment)
	d.registerNotify = make(chan struct {
		threadId uint64
		value    *fragment
	})
	tracer.RegisterChan(d.registerNotify, cap(d.registerNotify))
	d.cancelNotify = make(chan struct {
		threadId uint64
		value    error
	})
	tracer.RegisterChan(d.cancelNotify, cap(d.cancelNotify))
	d.lastOrdinalNotify = make(chan struct {
		threadId uint64
		value    int64
	})
	tracer.RegisterChan(d.lastOrdinalNotify, cap(d.lastOrdinalNotify))
	d.done = make(chan struct {
		threadId uint64
		value    struct{}
	})
	tracer.RegisterChan(d.done, cap(d.done))
}

func (d *defrag) nextFragment() *fragment {
	atomic.AddInt64(&d.lastAlloc, 1)
	f := fragment{ord: d.lastAlloc}

	return &f
}

func (d *defrag) cancel(err error) {
	myTIDCache := tracer.GetGID()
	tracer.PreSend(d.cancelNotify, "defrag.go:92", myTIDCache)
	d.cancelNotify <- struct {
		threadId uint64
		value    error
	}{myTIDCache, err}
	tracer.PostSend(d.cancelNotify, "defrag.go:92", myTIDCache)
}

func (d *defrag) WriteTo(dst io.Writer) (written int64, err error) {
	myTIDCache := tracer.GetGID()
	defer func() {
		tracer.PreClose(d.done, "defrag.go:104", myTIDCache)
		close(d.done)
		tracer.PostClose(d.done, "defrag.go:104", myTIDCache)
	}()

	if d.cancellation != nil {
		return d.written, d.cancellation
	}

	for {

		if d.lastWritten >= d.lastOrdinal && d.lastOrdinal > 0 {
			break
		}
		tracer.PreSelect(tracer.GetGID(), tracer.SelectEv{d.registerNotify, "?", "defrag.go:115"}, tracer.SelectEv{d.cancelNotify, "?", "defrag.go:115"},
			tracer.SelectEv{d.lastOrdinalNotify, "?", "defrag.go:115"})

		select {
		case tmp1 := <-d.registerNotify:
			tracer.PostRcv(d.registerNotify, "defrag.go:115", tmp1.threadId, myTIDCache)
			frag := tmp1.value

			next := d.lastWritten + 1
			if frag.ord == next {

				n, err := d.writeConsecutive(dst, frag)
				d.written += n
				if err != nil {
					return d.written, err
				}
			} else if frag.ord > next {
				d.future[frag.ord] = frag
			} else {
				return d.written, assertErrf(
					"Unexpected retrograde fragment %v, "+
						"expected at least %v",
					frag.ord, next)
			}

		case tmp2 := <-d.cancelNotify:
			tracer.PostRcv(d.cancelNotify, "defrag.go:115", tmp2.threadId, myTIDCache)
			d.cancellation = tmp2.value
			d.future = nil
			return d.written, d.cancellation

		case tmp3 := <-d.lastOrdinalNotify:
			tracer.PostRcv(d.lastOrdinalNotify, "defrag.go:115", tmp3.threadId, myTIDCache)
			d.lastOrdinal = tmp3.value
			continue
		}
	}

	return d.written, nil
}

func (d *defrag) setLast(lastOrdinal int64) {
	myTIDCache := tracer.GetGID()
	tracer.PreSelect(myTIDCache, tracer.SelectEv{d.lastOrdinalNotify, "!", "defrag.go:160"}, tracer.SelectEv{d.done, "?", "defrag.go:160"})
	select {
	case d.lastOrdinalNotify <- struct {
		threadId uint64
		value    int64
	}{myTIDCache, lastOrdinal}:
		tracer.PostSend(d.lastOrdinalNotify, "defrag.go:160", myTIDCache)
	case tmp := <-d.done:
		tracer.PostRcv(d.done, "defrag.go:160", tmp.threadId, myTIDCache)
	}
}

func (d *defrag) lastAllocated() int64 {
	return atomic.LoadInt64(&d.lastAlloc)
}

func (d *defrag) register(frag *fragment) {
	myTIDCache := tracer.GetGID()
	tracer.PreSend(d.registerNotify, "defrag.go:180", myTIDCache)
	d.registerNotify <- struct {
		threadId uint64
		value    *fragment
	}{myTIDCache, frag}
	tracer.PostSend(d.registerNotify, "defrag.go:180", myTIDCache)
}

func (d *defrag) writeConsecutive(dst io.Writer, start *fragment) (
	int64, error) {
	myTIDCache := tracer.GetGID()

	written, err := start.contents.WriteTo(dst)
	if err != nil {
		return int64(written), err
	}

	if err := start.contents.Close(); err != nil {
		return int64(written), err
	}

	d.lastWritten += 1

	for {
		tracer.PreSelect(myTIDCache, tracer.SelectEv{d.cancelNotify, "?", "defrag.go:196"}, tracer.SelectEv{nil, "?", "defrag.go:196"})

		select {
		case tmp9 := <-d.cancelNotify:
			d.cancellation = tmp9.value
			tracer.PostRcv(d.cancelNotify, "defrag.go:196", tmp9.threadId, myTIDCache)
			d.future = nil
			return 0, d.cancellation
		default:
			tracer.PostRcv(nil, "defrag.go:200", 0, myTIDCache)
		}

		next := d.lastWritten + 1
		if frag, ok := d.future[next]; ok {

			delete(d.future, next)
			n, err := frag.contents.WriteTo(dst)
			written += n
			defer frag.contents.Close()
			if err != nil {
				return int64(written), err
			}

			d.lastWritten = next
		} else {
			return int64(written), nil
		}
	}
}
