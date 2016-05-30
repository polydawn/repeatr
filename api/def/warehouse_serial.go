package def

import (
	"fmt"

	"github.com/ugorji/go/codec"
)

var _ codec.Selfer = &WarehouseCoords{}

func (wc WarehouseCoords) CodecEncodeSelf(c *codec.Encoder) {
	switch len(wc) {
	case 0:
		panic("impossible") // omitempty should already have struct
	case 1:
		c.MustEncode(wc[0])
	default:
		// note: no, we're not sorting:
		//  1) nah, don't care.  not part of conjecture anyway.
		//  2) order actually has a meaning: order in which to try.
		c.MustEncode([]string(wc))
	}
}

func (wc *WarehouseCoords) CodecDecodeSelf(c *codec.Decoder) {
	var val interface{}
	c.MustDecode(&val)
	switch val2 := val.(type) {
	case string:
		(*wc) = []string{val2}
	case []interface{}:
		(*wc) = coerceStringList(val2)
	default:
		panic(ConfigError.New("silo must be a string or list of strings"))
	}

}

func coerceStringList(x interface{}) []string {
	if w, ok := x.([]string); ok {
		return w
	}
	y, ok := x.([]interface{})
	if !ok {
		panic(fmt.Errorf(describe(x)))
	}
	z := make([]string, len(y))
	for i := range y {
		z[i], ok = y[i].(string)
		if !ok {
			panic(fmt.Errorf("%s at index %d", describe(x), i))
		}
	}
	return z
}
