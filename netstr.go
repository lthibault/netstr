package netstr

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"unsafe"

	"github.com/pkg/errors"
)

// Str is a netstring
type Str []byte

// String satisfies fmt.Stringer, returning the payload of the Str
func (str *Str) String() string { return string(*str) }

func (str Str) hdr() []byte {
	b := make([]byte, 8)
	i := binary.PutUvarint(b, uint64(len(str)))
	return b[:i]
}

// MarshalBinary implements encoding.BinaryMarshaller
func (str Str) MarshalBinary() ([]byte, error) {
	h := str.hdr()
	return append(h, str...), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaller
func (str *Str) UnmarshalBinary(b []byte) error {
	switch strLen, i := binary.Uvarint(b); {
	case i == 0:
		return errors.New("not enough data")
	case i < 0:
		return errors.New("invalid header:  exceeds 64 bits")
	case int(strLen)+i != len(b):
		return errors.Errorf("expected message of len %d, got %d", strLen, len(b)-i)
	default:
		*str = b[i:]
		return nil
	}
}

func readNetStr(r io.Reader) (Str, error) {
	var i int

	hdr := make([]byte, 8)
	br := bufio.NewReader(r)

	for {
		if i > 7 {
			return nil, errors.New("invalid header:  exceeds 64 bits")
		}

		b, err := br.ReadByte()
		if err != nil {
			return nil, errors.Wrapf(err, "read byte %d of header", i)
		}

		hdr[i] = b
		if *(*uint8)(unsafe.Pointer(&b))&uint8(128) == 0 {
			hdr = hdr[:i]
			break
		}

		i++
	}

	strLen, i := binary.Uvarint(hdr)
	if i != len(hdr) {
		panic("header parse succeeded, but binary decode failed")
	}

	var s Str = make([]byte, strLen)
	if _, err := io.ReadFull(r, s); err != nil {
		return nil, errors.Wrap(err, "read msg body")
	}

	return s, nil
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
		if _, e.err = io.Copy(e.w, bytes.NewBuffer(s.hdr())); e.err == nil {
			_, e.err = io.Copy(e.w, bytes.NewBuffer(s))
		}
	}

	return e.err
}

// A Decoder reads and decodes netstr values from an input stream.
type Decoder struct {
	err error
	r   io.Reader
}

// NewDecoder returns a new decoder that reads from r
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// Reset the decoder with a new Reader
func (d *Decoder) Reset(r io.Reader) {
	d.r = r
	d.err = nil
}

// Decode reads the next netstr-encoded value from its input and stores it in
// the netstr s
func (d *Decoder) Decode() (Str, error) {
	var s Str
	if d.err == nil {
		s, d.err = readNetStr(d.r)
	}

	return s, d.err
}
