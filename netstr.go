package netstr

import (
	"encoding/binary"
	"sync"
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
