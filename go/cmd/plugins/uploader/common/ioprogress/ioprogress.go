package ioprogress

import (
	"io"
	"strconv"
	"sync"
	"sync/atomic"
)

type Reader struct {
	count  atomic.Int64
	Total  int64
	reader io.Reader
	lock   *sync.Mutex
}

func NewReader(r io.Reader, total int64) *Reader {
	return &Reader{
		reader: r,
		Total:  total,
	}
}

func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if err == nil {
		r.count.Add(int64(n))
	}
	return
}

func (r *Reader) ChangeReader(reader io.Reader) {
	r.reader = reader
}
func (r *Reader) Progress() string {
	p := r.count.Load() * 100 / r.Total
	if p == 0 {
		return ""
	}
	if p > 100 {
		p = 100
	}
	return strconv.FormatInt(p, 10) + "%"
}
