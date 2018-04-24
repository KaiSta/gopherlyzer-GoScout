package pgzipInst

import (
	"bytes"
	"errors"
	"fmt"
	"hash"
	"io"
	"sync"
	"time"

	"../tracer"
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/crc32"
)

const (
	defaultBlockSize = 256 << 10
	tailSize         = 16384
	defaultBlocks    = 16
)

const (
	NoCompression       = flate.NoCompression
	BestSpeed           = flate.BestSpeed
	BestCompression     = flate.BestCompression
	DefaultCompression  = flate.DefaultCompression
	ConstantCompression = flate.ConstantCompression
	HuffmanOnly         = flate.HuffmanOnly
)

type Writer struct {
	Header
	w             io.Writer
	level         int
	wroteHeader   bool
	blockSize     int
	blocks        int
	currentBuffer []byte
	prevTail      []byte
	digest        hash.Hash32
	size          int
	closed        bool
	buf           [10]byte
	errMu         sync.RWMutex
	err           error
	pushedErr     chan struct {
		threadId uint64
		value    struct{}
	}
	results chan struct {
		threadId uint64
		value    result
	}
	dictFlatePool sync.Pool
	dstPool       sync.Pool
	wg            sync.WaitGroup
}

type result struct {
	result chan struct {
		threadId uint64
		value    []byte
	}
	notifyWritten chan struct {
		threadId uint64
		value    struct{}
	}
}

func (z *Writer) SetConcurrency(blockSize, blocks int) error {
	if blockSize <= tailSize {
		return fmt.Errorf("gzip: block size cannot be less than or equal to %d", tailSize)
	}
	if blocks <= 0 {
		return errors.New("gzip: blocks cannot be zero or less")
	}
	if blockSize == z.blockSize && blocks == z.blocks {
		return nil
	}
	z.blockSize = blockSize
	z.results = make(chan struct {
		threadId uint64
		value    result
	}, blocks)
	tracer.RegisterChan(z.results, cap(z.results))
	z.blocks = blocks
	z.dstPool = sync.Pool{New: func() interface{} { return make([]byte, 0, blockSize+(blockSize)>>4) }}
	return nil
}

func NewWriter(w io.Writer) *Writer {
	z, _ := NewWriterLevel(w, DefaultCompression)
	return z
}

func NewWriterLevel(w io.Writer, level int) (*Writer, error) {
	if level < ConstantCompression || level > BestCompression {
		return nil, fmt.Errorf("gzip: invalid compression level: %d", level)
	}
	z := new(Writer)
	z.SetConcurrency(defaultBlockSize, defaultBlocks)
	z.init(w, level)
	return z, nil
}

func (z *Writer) pushError(err error) {
	myTIDCache := tracer.GetGID()
	tracer.PreLock(&z.errMu, "gzip.go:127", myTIDCache)
	z.errMu.Lock()
	if z.err != nil {
		tracer.PostLock(&z.errMu, "gzip.go:129", myTIDCache)
		z.errMu.Unlock()
		return
	}
	z.err = err
	tracer.PreClose(z.pushedErr, "gzip.go:133", myTIDCache)
	close(z.pushedErr)
	tracer.PostClose(z.pushedErr, "gzip.go:133", myTIDCache)
	tracer.PostLock(&z.errMu, "gzip.go:134", myTIDCache)
	z.errMu.Unlock()
}

func (z *Writer) init(w io.Writer, level int) {
	z.wg.Wait()
	digest := z.digest
	if digest != nil {
		digest.Reset()
	} else {
		digest = crc32.NewIEEE()
	}
	z.Header = Header{OS: 255}
	z.w = w
	z.level = level
	z.digest = digest
	z.pushedErr = make(chan struct {
		threadId uint64
		value    struct{}
	}, 0)
	tracer.RegisterChan(z.pushedErr, cap(z.pushedErr))
	z.results = make(chan struct {
		threadId uint64
		value    result
	}, z.blocks)
	tracer.RegisterChan(z.results, cap(z.results))
	z.err = nil
	z.closed = false
	z.Comment = ""
	z.Extra = nil
	z.ModTime = time.Time{}
	z.wroteHeader = false
	z.currentBuffer = nil
	z.buf = [10]byte{}
	z.prevTail = nil
	z.size = 0
	if z.dictFlatePool.New == nil {
		z.dictFlatePool.New = func() interface{} {
			f, _ := flate.NewWriterDict(w, level, nil)
			return f
		}
	}
}

func (z *Writer) Reset(w io.Writer) {
	myTIDCache := tracer.GetGID()
	if z.results != nil && !z.closed {
		tracer.PreClose(z.results, "gzip.go:175", myTIDCache)
		close(z.results)
		tracer.PostClose(z.results, "gzip.go:175", myTIDCache)
	}
	z.SetConcurrency(defaultBlockSize, defaultBlocks)
	z.init(w, z.level)
}

func put2(p []byte, v uint16) {
	p[0] = uint8(v >> 0)
	p[1] = uint8(v >> 8)
}

func put4(p []byte, v uint32) {
	p[0] = uint8(v >> 0)
	p[1] = uint8(v >> 8)
	p[2] = uint8(v >> 16)
	p[3] = uint8(v >> 24)
}

func (z *Writer) writeBytes(b []byte) error {
	if len(b) > 0xffff {
		return errors.New("gzip.Write: Extra data is too large")
	}
	put2(z.buf[0:2], uint16(len(b)))
	_, err := z.w.Write(z.buf[0:2])
	if err != nil {
		return err
	}
	_, err = z.w.Write(b)
	return err
}

func (z *Writer) writeString(s string) (err error) {

	needconv := false
	for _, v := range s {
		if v == 0 || v > 0xff {
			return errors.New("gzip.Write: non-Latin-1 header string")
		}
		if v > 0x7f {
			needconv = true
		}
	}
	if needconv {
		b := make([]byte, 0, len(s))
		for _, v := range s {
			b = append(b, byte(v))
		}
		_, err = z.w.Write(b)
	} else {
		_, err = io.WriteString(z.w, s)
	}
	if err != nil {
		return err
	}

	z.buf[0] = 0
	_, err = z.w.Write(z.buf[0:1])
	return err
}

func (z *Writer) compressCurrent(flush bool) {
	myTIDCache := tracer.GetGID()
	r := result{}
	r.result = make(chan struct {
		threadId uint64
		value    []byte
	}, 1)
	tracer.RegisterChan(r.result, cap(r.result))
	r.notifyWritten = make(chan struct {
		threadId uint64
		value    struct{}
	}, 0)
	tracer.RegisterChan(r.notifyWritten, cap(r.notifyWritten))
	tracer.PreSelect(myTIDCache, tracer.SelectEv{z.results, "!", "gzip.go:245"}, tracer.SelectEv{z.pushedErr, "?", "gzip.go:245"})
	select {
	case z.results <- struct {
		threadId uint64
		value    result
	}{myTIDCache, r}:
		tracer.PostSend(z.results, "gzip.go:245", myTIDCache)
	case tmp := <-z.pushedErr:
		tracer.PostRcv(z.results, "gzip.go:245", tmp.threadId, myTIDCache)
		return
	}

	c := z.currentBuffer
	if len(c) > z.blockSize*2 {
		c = c[:z.blockSize]
		z.wg.Add(1)
		tmp2 := tracer.GetWaitSigID()
		tracer.Signal(tmp2, myTIDCache)
		go func() {
			tracer.RegisterThread("z.compressBlock0")
			tracer.Wait(tmp2, tracer.GetGID())
			z.compressBlock(c, z.prevTail, r, false)
		}()
		z.prevTail = c[len(c)-tailSize:]
		z.currentBuffer = z.currentBuffer[z.blockSize:]
		z.compressCurrent(flush)

		return
	}

	z.wg.Add(1)
	tmp3 := tracer.GetWaitSigID()
	tracer.Signal(tmp3, myTIDCache)
	go func() {
		tracer.RegisterThread("z.compressBlock1")
		tracer.Wait(tmp3, tracer.GetGID())
		z.compressBlock(c, z.prevTail, r, z.closed)
	}()
	if len(c) > tailSize {
		z.prevTail = c[len(c)-tailSize:]
	} else {
		z.prevTail = nil
	}
	z.currentBuffer = z.dstPool.Get().([]byte)
	z.currentBuffer = z.currentBuffer[:0]

	if flush {
		myTIDCache := tracer.GetGID()
		tracer.PreRcv(r.notifyWritten, "gzip.go:276", myTIDCache)
		tmp4 := <-r.notifyWritten
		tracer.PostRcv(r.notifyWritten, "gzip.go:276", tmp4.threadId, myTIDCache)
	}
}

func (z *Writer) checkError() error {
	myTIDCache := tracer.GetGID()
	tracer.RPreLock(&z.errMu, "gzip.go:283", myTIDCache)
	z.errMu.RLock()
	err := z.err
	tracer.RPostLock(&z.errMu, "gzip.go:285", myTIDCache)
	z.errMu.RUnlock()
	return err
}

func (z *Writer) Write(p []byte) (int, error) {
	myTIDCache := tracer.GetGID()
	if err := z.checkError(); err != nil {
		return 0, err
	}

	if !z.wroteHeader {
		z.wroteHeader = true
		z.buf[0] = gzipID1
		z.buf[1] = gzipID2
		z.buf[2] = gzipDeflate
		z.buf[3] = 0
		if z.Extra != nil {
			z.buf[3] |= 0x04
		}
		if z.Name != "" {
			z.buf[3] |= 0x08
		}
		if z.Comment != "" {
			z.buf[3] |= 0x10
		}
		put4(z.buf[4:8], uint32(z.ModTime.Unix()))
		if z.level == BestCompression {
			z.buf[8] = 2
		} else if z.level == BestSpeed {
			z.buf[8] = 4
		} else {
			z.buf[8] = 0
		}
		z.buf[9] = z.OS
		var n int
		var err error
		n, err = z.w.Write(z.buf[0:10])
		if err != nil {
			z.pushError(err)
			return n, err
		}
		if z.Extra != nil {
			err = z.writeBytes(z.Extra)
			if err != nil {
				z.pushError(err)
				return n, err
			}
		}
		if z.Name != "" {
			err = z.writeString(z.Name)
			if err != nil {
				z.pushError(err)
				return n, err
			}
		}
		if z.Comment != "" {
			err = z.writeString(z.Comment)
			if err != nil {
				z.pushError(err)
				return n, err
			}
		}
		tmp5 := tracer.GetWaitSigID()
		tracer.Signal(tmp5, myTIDCache)

		go func() {
			tracer.RegisterThread("fun2")
			tracer.Wait(tmp5, tracer.GetGID())
			myTIDCache := tracer.GetGID()
			listen := z.results
			for {
				tracer.PreRcv(z.results, "gzip.go:378", myTIDCache)
				tmp6, ok := <-listen
				tracer.PostRcv(z.results, "gzip.go:378", tmp6.threadId, myTIDCache)

				r := tmp6.value

				if !ok {
					return
				}

				tracer.PreRcv(r.result, "gzip.go:386", myTIDCache)
				tmp7 := <-r.result
				tracer.PostRcv(r.result, "gzip.go:386", tmp7.threadId, myTIDCache)
				buf := tmp7.value
				n, err := z.w.Write(buf)
				if err != nil {
					z.pushError(err)
					tracer.PreClose(r.notifyWritten, "gzip.go:371", myTIDCache)
					close(r.notifyWritten)
					tracer.PostClose(r.notifyWritten, "gzip.go:371", myTIDCache)
					return
				}
				if n != len(buf) {
					z.pushError(fmt.Errorf("gzip: short write %d should be %d", n, len(buf)))
					tracer.PreClose(r.notifyWritten, "gzip.go:376", myTIDCache)
					close(r.notifyWritten)
					tracer.PostClose(r.notifyWritten, "gzip.go:376", myTIDCache)
					return
				}
				z.dstPool.Put(buf)
				tracer.PreClose(r.notifyWritten, "gzip.go:380", myTIDCache)
				close(r.notifyWritten)
				tracer.PostClose(r.notifyWritten, "gzip.go:380", myTIDCache)
			}
		}()

		z.currentBuffer = make([]byte, 0, z.blockSize)
	}
	q := p
	for len(q) > 0 {
		length := len(q)
		if length+len(z.currentBuffer) > z.blockSize {
			length = z.blockSize - len(z.currentBuffer)
		}
		z.digest.Write(q[:length])
		z.currentBuffer = append(z.currentBuffer, q[:length]...)
		if len(z.currentBuffer) >= z.blockSize {
			z.compressCurrent(false)
			if err := z.checkError(); err != nil {
				return len(p) - len(q) - length, err
			}
		}
		z.size += length
		q = q[length:]
	}
	return len(p), z.checkError()
}

func (z *Writer) compressBlock(p, prevTail []byte, r result, closed bool) {
	myTIDCache := tracer.GetGID()
	defer func() {
		close(r.result)
		z.wg.Done()
	}()
	buf := z.dstPool.Get().([]byte)
	dest := bytes.NewBuffer(buf[:0])

	compressor := z.dictFlatePool.Get().(*flate.Writer)
	compressor.ResetDict(dest, prevTail)
	compressor.Write(p)

	err := compressor.Flush()
	if err != nil {
		z.pushError(err)
		return
	}
	if closed {
		err = compressor.Close()
		if err != nil {
			z.pushError(err)
			return
		}
	}
	z.dictFlatePool.Put(compressor)

	buf = dest.Bytes()

	tracer.PreSend(r.result, "gzip:503", myTIDCache)
	r.result <- struct {
		threadId uint64
		value    []byte
	}{myTIDCache, buf}
	tracer.PostSend(r.result, "gzip:503", myTIDCache)
}

func (z *Writer) Flush() error {
	if err := z.checkError(); err != nil {
		return err
	}
	if z.closed {
		return nil
	}
	if !z.wroteHeader {
		_, err := z.Write(nil)
		if err != nil {
			return err
		}
	}

	z.compressCurrent(true)

	return z.checkError()
}

func (z *Writer) UncompressedSize() int {
	return z.size
}

func (z *Writer) Close() error {
	myTIDCache := tracer.GetGID()
	if err := z.checkError(); err != nil {
		return err
	}
	if z.closed {
		return nil
	}

	z.closed = true
	if !z.wroteHeader {
		z.Write(nil)
		if err := z.checkError(); err != nil {
			return err
		}
	}
	z.compressCurrent(true)
	if err := z.checkError(); err != nil {
		return err
	}
	tracer.PreClose(z.results, "gzip.go:492", myTIDCache)
	close(z.results)
	tracer.PostClose(z.results, "gzip.go:492", myTIDCache)
	put4(z.buf[0:4], z.digest.Sum32())
	put4(z.buf[4:8], uint32(z.size))
	_, err := z.w.Write(z.buf[0:8])
	if err != nil {
		z.pushError(err)
		return err
	}
	return nil
}
