package netstr

import (
	"encoding/binary"
	"testing"
)

func TestRepr(t *testing.T) {
	var s = "hello"
	var str Str = []byte(s)

	t.Run("Bytes", func(t *testing.T) {
		b := str.Bytes()
		expected := []byte{5, 0, 0, 0, 0}
		for i := 0; i < binary.MaxVarintLen32; i++ {
			if b[i] != expected[i] {
				t.Error("improperly encoded header")
			}
		}
	})

	t.Run("String", func(t *testing.T) {
		if str.String() != s {
			t.Errorf("expected \"hello\", got %s", str.String())
		}
	})
}
