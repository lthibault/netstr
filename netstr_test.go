package netstr

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const multiByte string = `Lorem ipsum dolor sit amet, consectetur adipiscing
elit. Praesent ut ultrices metus. Donec id euismod arcu. Maecenas id enim 
rhoncus, bibendum urna vel, malesuada libero. Proin venenatis nibh vitae euismod
viverra. Curabitur molestie mi nulla, semper amet.`

func TestStr(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		assert.Equal(t, "hello, world", Str("hello, world").String())
	})

	t.Run("Header", func(t *testing.T) {
		t.Run("SingleByte", func(t *testing.T) {
			b := make([]byte, binary.MaxVarintLen64)
			assert.Equal(
				t,
				Str("foo").ByteLen(),
				b[:binary.PutUvarint(b, uint64(3))],
			)
		})

		t.Run("MultiByte", func(t *testing.T) {
			b := make([]byte, binary.MaxVarintLen64)
			assert.Equal(
				t,
				Str(multiByte).ByteLen(),
				b[:binary.PutUvarint(b, uint64(len(multiByte)))],
			)
		})
	})

	t.Run("Codec", func(t *testing.T) {
		str := Str("hello, world")
		t.Run("MarshalBinary", func(t *testing.T) {
			b, err := str.MarshalBinary()
			assert.NoError(t, err)
			assert.Equal(
				t,
				append(str.ByteLen(), str...),
				b,
			)
		})

		t.Run("UnmarshalBinary", func(t *testing.T) {
			var str Str

			b := make([]byte, binary.MaxVarintLen64)
			b = append(b[:binary.PutUvarint(b, 3)], []byte("foo")...)

			assert.NoError(t, str.UnmarshalBinary(b))
			assert.Equal(t, "foo", str.String())
		})
	})
}

func TestEncoder(t *testing.T) {
	buf := new(bytes.Buffer)
	enc := NewEncoder(buf)

	t.Run("Encode", func(t *testing.T) {
		assert.NoError(t, enc.Encode([]byte("hello")))
		assert.Len(t, buf.Bytes(), 6)
		assert.Equal(t, []byte{0x5, 'h', 'e', 'l', 'l', 'o'}, buf.Bytes())
	})

	t.Run("Reset", func(t *testing.T) {
		b := new(bytes.Buffer)
		enc.err = errors.New("test")
		enc.Reset(b)
		assert.Equal(t, b, enc.w)
		assert.NoError(t, enc.err)
	})
}

func TestSplit(t *testing.T) {

	// NOTE:  bufio.Scanner interprets output from bufio.SplitFunc in a very
	//		  specific way.  https://golang.org/pkg/bufio/#SplitFunc

	var (
		advance int
		token   []byte
		err     error
	)

	t.Run("Header", func(t *testing.T) {
		t.Run("Underrun", func(t *testing.T) {
			b := make([]byte, binary.MaxVarintLen64)
			b = b[:binary.PutUvarint(b, 512)]

			t.Run("EOF", func(t *testing.T) {
				advance, token, err = Split(b[:1], true)
				assert.Zero(t, advance)
				assert.Nil(t, token)
				assert.Error(t, err)
			})

			t.Run("ReadMore", func(t *testing.T) {
				advance, token, err = Split(b[:1], false)
				assert.Zero(t, advance)
				assert.Nil(t, token)
				assert.NoError(t, err)
			})
		})

		// t.Run("Overrun", func(t *testing.T) {
		// 	// TODO
		// })
	})

	t.Run("Body", func(t *testing.T) {
		b := make([]byte, binary.MaxVarintLen64)
		b = b[:binary.PutUvarint(b, 12)]

		t.Run("Underrun", func(t *testing.T) {
			t.Run("EOF", func(t *testing.T) {
				advance, token, err = Split(append(b, []byte("hello")...), true)
				assert.Zero(t, advance)
				assert.Nil(t, token)
				assert.Error(t, err)
			})

			t.Run("ReadMore", func(t *testing.T) {
				advance, token, err = Split(append(b, []byte("hello")...), false)
				assert.Zero(t, advance)
				assert.Nil(t, token)
				assert.NoError(t, err)
			})
		})

		t.Run("Success", func(t *testing.T) {
			advance, token, err = Split(append(b, []byte("hello, world. foo bar.")...), true)
			assert.Equal(t, 13, advance) // 12 + 1 varint
			assert.Equal(t, "hello, world", string(token))
			assert.NoError(t, err)
		})
	})
}

func TestDecoder(t *testing.T) {
	var dec *Decoder

	t.Run("Decode", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			b := make([]byte, binary.MaxVarintLen64)
			b = append(b[:binary.PutUvarint(b, 3)], []byte("foo")...)

			dec = NewDecoder(bytes.NewBuffer(b))

			str, err := dec.Decode()
			assert.NoError(t, err)
			assert.Equal(t, "foo", str.String())
			assert.False(t, dec.eof)
		})

		t.Run("EOF", func(t *testing.T) {
			_, err := dec.Decode()
			assert.Equal(t, err, io.EOF)
			assert.Error(t, err)
			assert.True(t, dec.eof)
		})
	})

	t.Run("Reset", func(t *testing.T) {
		buf := new(bytes.Buffer)
		dec.Reset(buf)
		assert.False(t, dec.eof)
		assert.NoError(t, dec.scanner.Err())
	})
}
