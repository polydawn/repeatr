package def

import (
	"bytes"
	"sort"

	"github.com/ugorji/go/codec"
)

var _ codec.Selfer = &InputGroup{}

func (ig InputGroup) CodecEncodeSelf(c *codec.Encoder) {
	c.Encode(ig.asMappySlice())
}

func (ig *InputGroup) CodecDecodeSelf(c *codec.Decoder) {
	// I'd love to just punt to the defaults, but the `Selfer` interface doesn't come in half.
	// SO here's a ridiculous indirection to prance around infinite recursion.
	c.MustDecode((*map[string]*Input)(ig))
}

func (ig InputGroup) asMappySlice() codec.MapBySlice {
	keys := make([]string, len(ig))
	var i int
	for k, _ := range ig {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	val := make(mappySlice, len(ig)*2)
	i = 0
	for _, k := range keys {
		val[i] = k
		i++
		val[i] = ig[k]
		i++
	}
	return val
}

func (i Input) String() string {
	var buf bytes.Buffer
	codec.NewEncoder(&buf, &codec.JsonHandle{}).Encode(i)
	return buf.String()
}
