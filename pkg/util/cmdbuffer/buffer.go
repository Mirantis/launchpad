// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// copied from https://cs.opensource.google/go/x/crypto/+/master:ssh/buffer.go

package cmdbuffer

import (
	"io"
	"sync"
)

// buffer provides a linked list buffer for data exchange
// between producer and consumer. Theoretically the buffer is
// of unlimited capacity as it does no allocation of its own.
type buffer struct {
	// protects concurrent access to head, tail and closed
	*sync.Cond

	head *element // the buffer that will be read first
	tail *element // the buffer that will be read last

	closed bool
}

// An element represents a single link in a linked list.
type element struct {
	buf  []byte
	next *element
}

// NewBuffer returns an empty buffer that does not EOF until it is closed.
func NewBuffer() *buffer {
	e := new(element)
	b := &buffer{
		Cond: sync.NewCond(new(sync.Mutex)),
		head: e,
		tail: e,
	}
	return b
}

// EOF closes the buffer. Reads from the buffer once all
// the data has been consumed will receive io.EOF.
func (b *buffer) EOF() {
	b.Cond.L.Lock()
	b.closed = true
	b.Cond.Signal()
	b.Cond.L.Unlock()
}

// Write makes buf available for Read to receive.
// buf must not be modified after the call to write.
func (b *buffer) Write(buf []byte) (int, error) {
	b.Cond.L.Lock()
	e := &element{buf: buf}
	b.tail.next = e
	b.tail = e
	b.Cond.Signal()
	b.Cond.L.Unlock()
	return len(buf), nil
}

// Read reads data from the internal buffer in buf.  Reads will block
// if no data is available, or until the buffer is closed.
func (b *buffer) Read(buf []byte) (n int, err error) {
	b.Cond.L.Lock()
	defer b.Cond.L.Unlock()

	for len(buf) > 0 {
		// if there is data in b.head, copy it
		if len(b.head.buf) > 0 {
			r := copy(buf, b.head.buf)
			buf, b.head.buf = buf[r:], b.head.buf[r:]
			n += r
			continue
		}
		// if there is a next buffer, make it the head
		if len(b.head.buf) == 0 && b.head != b.tail {
			b.head = b.head.next
			continue
		}

		// if at least one byte has been copied, return
		if n > 0 {
			break
		}

		// if nothing was read, and there is nothing outstanding
		// check to see if the buffer is closed.
		if b.closed {
			err = io.EOF
			break
		}
		// out of buffers, wait for producer
		b.Cond.Wait()
	}
	return n, err
}
