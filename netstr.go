package netstr

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"unsafe"
)

const prefixSize = binary.MaxVarintLen32

var prefixPool = &sync.Pool{New: func() interface{} {
	return make([]byte, prefixSize)
}}

// Str is a netstring
type Str []byte

// String satisfies fmt.Stringer, returning the payload of the Str
func (str *Str) String() string { return string(*str) }

// Bytes returns netstring-encoded array of bytes
func (str *Str) Bytes() (b []byte) {
	buf := prefixPool.Get().([]byte)
	binary.PutUvarint(buf, uint64(len(*str)))
	b = append(buf, *str...)
	prefixPool.Put(buf)
	return
}

// Encoder encodes byte arrays into netstrings and writes them to a byte-stream
type Encoder struct {
	w   io.Writer
	err error
}

// Reset the encoder with a new writer.  Useful for encoder pools.
func (e *Encoder) Reset(w io.Writer) {
	e.w = w
	e.err = nil
}

// Feed bytes into the stream, netstring-encoding them on the fly
func (e *Encoder) Feed(b []byte) (err error) {
	str := (*Str)(unsafe.Pointer(&b))
	if _, err = e.w.Write(str.Bytes()); err != nil {
		e.err = err
	}

	return
}

// Decoder decodes netstrings from a byte-streams
type Decoder struct {
	r   io.Reader
	err error
}

// Reset the decoder with a new reader.  Useful for decoder pools.
func (d *Decoder) Reset(r io.Reader) {
	d.r = r
	d.err = nil
}

// Next message
func (d *Decoder) Next() ([]byte, error) {
	if d.err != nil {
		return nil, d.err
	}

	pfx := prefixPool.Get().([]byte)
	defer prefixPool.Put(pfx)

	var err error
	if _, err = io.ReadFull(d.r, pfx); err != nil {
		d.err = err
		return nil, err
	}

	x, n := binary.Uvarint(pfx)
	if n == 0 {
		return nil, errors.New("buffer too small")
	} else if n < 0 {
		return nil, errors.New("overflow")
	}

	b := make([]byte, x)
	if _, err = io.ReadFull(d.r, b); err != nil {
		d.err = err
		return nil, err
	}

	return b, nil
}
