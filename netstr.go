package netstr

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"sync"

	"github.com/pkg/errors"
)

var pool = sync.Pool{New: func() interface{} { return make([]byte, 1) }}

// Str is a netstring
type Str []byte

// String satisfies fmt.Stringer, returning the payload of the Str
func (str Str) String() string { return string(str) }

// ByteLen is a binary representation of the string length
func (str Str) ByteLen() []byte {
	b := make([]byte, binary.MaxVarintLen64)
	return b[:binary.PutUvarint(b, uint64(len(str)))]
}

// Encode returns the netstr-encoded data
func (str Str) Encode() []byte {
	return append(str.ByteLen(), str...)
}

// MarshalBinary is a wrapper around Encode that implements
// encoding.BinaryMarshaller.  `err` is always nil.
func (str Str) MarshalBinary() ([]byte, error) {
	return str.Encode(), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaller
func (str *Str) UnmarshalBinary(b []byte) (err error) {
	var advance int
	if advance, *str, err = Split(b, true); advance != len(b) {
		err = errors.Errorf("expected message of len %d, got %d", len(b), advance)
	}
	return
}

// An Encoder writes netstr values to an output stream.
type Encoder struct {
	err error
	w   io.Writer
}

// NewEncoder returns a new encoder that writes to w
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Reset the encoder with a new Writer
func (e *Encoder) Reset(w io.Writer) {
	e.w = w
	e.err = nil
}

// Encode writes the netstr encoding of s to the stream
func (e *Encoder) Encode(s Str) error {
	if e.err == nil {
		if _, e.err = io.Copy(e.w, bytes.NewBuffer(s.ByteLen())); e.err == nil {
			_, e.err = io.Copy(e.w, bytes.NewBuffer(s))
		}
	}

	return e.err
}

// A Decoder reads and decodes netstr values from an input stream.
type Decoder struct {
	eof     bool
	scanner *bufio.Scanner
}

// NewDecoder returns a new decoder that reads from r
func NewDecoder(r io.Reader) (dec *Decoder) {
	dec = new(Decoder)
	dec.Reset(r)
	return
}

// Reset the decoder with a new Reader
func (d *Decoder) Reset(r io.Reader) {
	d.scanner = bufio.NewScanner(r)
	d.scanner.Split(Split)
	d.eof = false
}

// Decode reads the next netstr-encoded value from its input and stores it in
// the netstr s
func (d *Decoder) Decode() (Str, error) {
	if d.scanner.Err() == nil && !d.eof {
		d.eof = !d.scanner.Scan()
	}

	if d.eof {
		return nil, io.EOF
	}

	return d.scanner.Bytes(), d.scanner.Err()
}

// Split is a bufio.SplitFunc that allows a scanner to tokenize a stream into
// raw (encoded) netstrs.
func Split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	length, i := binary.Uvarint(data)
	switch {
	case i == 0:
		if atEOF {
			return 0, nil, errors.New("EOF occurred before end of header")
		}
		return 0, nil, nil // ask for more data
	case i < 0:
		return 0, nil, errors.New("invalid header:  exceeds 64 bits")
	}

	if len(data) < int(length)-i {
		if atEOF {
			return 0, nil, errors.New("EOF occurred before end of body")
		}

		return 0, nil, nil
	}

	return i + int(length), data[i : i+int(length)], nil
}
