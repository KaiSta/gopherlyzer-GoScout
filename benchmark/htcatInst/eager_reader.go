package htcat

import (
	"io"
	"sync"

	"../tracer"
)

type eagerReader struct {
	closeNotify chan struct {
		threadId uint64
		value    struct{}
	}
	rc io.ReadCloser

	buf   []byte
	more  *sync.Cond
	begin int
	end   int

	lastErr error
}

func newEagerReader(r io.ReadCloser, bufSz int64) *eagerReader {
	myTIDCache := tracer.GetGID()
	er := eagerReader{
		rc:  r,
		buf: make([]byte, bufSz, bufSz),
	}
	er.closeNotify = make(chan struct {
		threadId uint64
		value    struct{}
	})
	tracer.RegisterChan(er.closeNotify, cap(er.closeNotify))

	er.more = sync.NewCond(new(sync.Mutex))
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, myTIDCache)

	go func() {
		tracer.RegisterThread("er.buffer0")
		tracer.Wait(tmp1, tracer.GetGID())
		er.buffer()
	}()

	return &er
}

func (er *eagerReader) buffer() {
	myTIDCache := tracer.GetGID()
	for er.lastErr == nil && er.end != len(er.buf) {
		var n int
		tracer.PreLock(&er.more.L, "eager_reader.go:38", myTIDCache)
		er.more.L.Lock()
		n, er.lastErr = er.rc.Read(er.buf[er.end:])
		er.end += n

		er.more.Broadcast()
		tracer.PostLock(&er.more.L, "eager_reader.go:43", myTIDCache)
		er.more.L.Unlock()
	}
}

func (er *eagerReader) writeOnce(dst io.Writer) (int64, error) {
	myTIDCache := tracer.GetGID()
	tracer.PreLock(&er.more.L, "eager_reader.go:52", myTIDCache)
	er.more.L.Lock()
	defer func() {
		tracer.PostLock(&er.more.L, "eager_reader.go:53", myTIDCache)
		er.more.L.Unlock()
	}()

	for er.begin == er.end {
		if er.lastErr != nil {
			return 0, er.lastErr
		}

		if er.begin == len(er.buf) {
			return 0, io.EOF
		}

		er.more.Wait()
	}

	n, err := dst.Write(er.buf[er.begin:er.end])
	er.begin += n
	return int64(n), err
}

func (er *eagerReader) WriteTo(dst io.Writer) (int64, error) {
	var written int64

	for {
		n, err := er.writeOnce(dst)
		written += n
		switch err {
		case io.EOF:

			return 0, nil
		case nil:

			continue
		default:

			return written, err
		}
	}
}

func (er *eagerReader) Close() error {
	err := er.rc.Close()
	tracer.PreSend(er.closeNotify, "eager_reader.go:100", tracer.GetGID())
	er.closeNotify <- struct {
		threadId uint64
		value    struct{}
	}{tracer.GetGID(), struct{}{}}
	tracer.PostSend(er.closeNotify, "eager_reader.go:100", tracer.GetGID())
	return err

}

func (er *eagerReader) WaitClosed() {
	tracer.PreRcv(er.closeNotify, "eager_reader.go:105", tracer.GetGID())
	tmp := <-er.closeNotify
	tracer.PostRcv(er.closeNotify, "eager_reader.go:105", tmp.threadId, tracer.GetGID())
}
