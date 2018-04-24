package htcat

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"../tracer"
)

const (
	_        = iota
	kB int64 = 1 << (10 * iota)
	mB
	gB
	tB
	pB
	eB
)

type HtCat struct {
	io.WriterTo
	d  defrag
	u  *url.URL
	cl *http.Client

	httpFragGenMu sync.Mutex
	hfg           httpFragGen
}

type HttpStatusError struct {
	error
	Status string
}

func (cat *HtCat) startup(parallelism int) {
	myTIDCache := tracer.GetGID()
	req := http.Request{
		Method:     "GET",
		URL:        cat.u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Body:       nil,
		Host:       cat.u.Host,
	}

	resp, err := cat.cl.Do(&req)
	if err != nil {
		tmp1 := tracer.GetWaitSigID()
		tracer.Signal(tmp1, myTIDCache)
		go func() {
			tracer.RegisterThread("cat.d.cancel0")
			tracer.Wait(tmp1, tracer.GetGID())
			cat.d.cancel(err)
		}()
		return
	}

	if resp.StatusCode != 200 {
		err = HttpStatusError{
			error: fmt.Errorf(
				"Expected HTTP Status 200, received: %q",
				resp.Status),
			Status: resp.Status}
		tmp2 := tracer.GetWaitSigID()
		tracer.Signal(tmp2, myTIDCache)
		go func() {
			tracer.RegisterThread("cat.d.cancel1")
			tracer.Wait(tmp2, tracer.GetGID())
			cat.d.cancel(err)
		}()
		return
	}

	l := resp.Header.Get("Content-Length")

	noParallel := func(wtc writerToCloser) {
		f := cat.d.nextFragment()
		cat.d.setLast(cat.d.lastAllocated())
		f.contents = wtc
		cat.d.register(f)
	}

	if l == "" {
		tmp3 := tracer.GetWaitSigID()
		tracer.Signal(tmp3, myTIDCache)

		go func() {
			tracer.RegisterThread("noParallel2")
			tracer.Wait(tmp3, tracer.GetGID())
			noParallel(struct {
				io.WriterTo
				io.Closer
			}{
				WriterTo: bufio.NewReader(resp.Body),
				Closer:   resp.Body,
			})
		}()
		return
	}

	length, err := strconv.ParseInt(l, 10, 64)
	if err != nil {
		tmp4 := tracer.GetWaitSigID()
		tracer.Signal(tmp4, myTIDCache)

		go func() {
			tracer.RegisterThread("cat.d.cancel3")
			tracer.Wait(tmp4, tracer.GetGID())
			cat.d.cancel(err)
		}()
		return
	}

	cat.hfg.totalSize = length
	cat.hfg.targetFragSize = 1 + ((length - 1) / int64(parallelism))
	if cat.hfg.targetFragSize > 20*mB {
		cat.hfg.targetFragSize = 20 * mB
	}

	if cat.hfg.targetFragSize < 1*mB {
		cat.hfg.curPos = cat.hfg.totalSize
		er := newEagerReader(resp.Body, cat.hfg.totalSize)
		tmp5 := tracer.GetWaitSigID()
		tracer.Signal(tmp5, myTIDCache)
		go func() {
			tracer.RegisterThread("noParallel4")
			tracer.Wait(tmp5, tracer.GetGID())
			noParallel(er)
		}()
		tmp6 := tracer.GetWaitSigID()
		tracer.Signal(tmp6, myTIDCache)
		go func() {
			tracer.RegisterThread("er.WaitClosed5")
			tracer.Wait(tmp6, tracer.GetGID())
			er.WaitClosed()
		}()
		return
	}

	hf := cat.nextFragment()
	tmp7 := tracer.GetWaitSigID()
	tracer.Signal(tmp7, myTIDCache)
	go func() {
		tracer.RegisterThread("fun6")
		tracer.Wait(tmp7, tracer.GetGID())
		er := newEagerReader(
			struct {
				io.Reader
				io.Closer
			}{
				Reader: io.LimitReader(resp.Body, hf.size),
				Closer: resp.Body,
			},
			hf.size)

		hf.fragment.contents = er
		cat.d.register(hf.fragment)
		er.WaitClosed()

		cat.get()
	}()

}

func New(client *http.Client, u *url.URL, parallelism int) *HtCat {
	myTIDCache := tracer.GetGID()
	cat := HtCat{
		u:  u,
		cl: client,
	}

	cat.d.initDefrag()
	cat.WriterTo = &cat.d
	cat.startup(parallelism)

	if cat.hfg.curPos == cat.hfg.totalSize {
		return &cat
	}

	for i := 1; i < parallelism; i += 1 {
		tmp8 := tracer.GetWaitSigID()
		tracer.Signal(tmp8, myTIDCache)
		go func() {
			tracer.RegisterThread("cat.get7")
			tracer.Wait(tmp8, tracer.GetGID())
			cat.get()
		}()
	}

	return &cat
}

func (cat *HtCat) nextFragment() *httpFrag {
	myTIDCache := tracer.GetGID()
	tracer.PreLock(&cat.httpFragGenMu, "http.go:173", myTIDCache)
	cat.httpFragGenMu.Lock()
	defer func() {
		tracer.PostLock(&cat.httpFragGenMu, "http.go:174", myTIDCache)
		cat.httpFragGenMu.Unlock()
	}()

	var hf *httpFrag

	if cat.hfg.hasNext() {
		f := cat.d.nextFragment()
		hf = cat.hfg.nextFragment(f)
	} else {
		cat.d.setLast(cat.d.lastAllocated())
	}

	return hf
}

func (cat *HtCat) get() {
	myTIDCache := tracer.GetGID()
	for {
		hf := cat.nextFragment()
		if hf == nil {
			return
		}

		req := http.Request{
			Method:     "GET",
			URL:        cat.u,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     hf.header,
			Body:       nil,
			Host:       cat.u.Host,
		}

		resp, err := cat.cl.Do(&req)
		if err != nil {
			cat.d.cancel(err)
			return
		}

		if !(resp.StatusCode == 206 || resp.StatusCode == 200) {
			err = HttpStatusError{
				error: fmt.Errorf("Expected HTTP Status "+
					"206 or 200, received: %q",
					resp.Status),
				Status: resp.Status}
			tmp9 := tracer.GetWaitSigID()
			tracer.Signal(tmp9, myTIDCache)
			go func() {
				tracer.RegisterThread("cat.d.cancel8")
				tracer.Wait(tmp9, tracer.GetGID())
				cat.d.cancel(err)
			}()
			return
		}

		er := newEagerReader(resp.Body, hf.size)
		hf.fragment.contents = er
		cat.d.register(hf.fragment)
		er.WaitClosed()
	}
}
