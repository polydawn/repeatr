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
	// Now go back over the struct and fill in MountPath as needed from the map keys.
	for k, v := range *ig {
		if v.MountPath == "" {
			v.MountPath = k
		}
	}
}

func (mp InputGroup) asMappySlice() codec.MapBySlice {
	keys := make([]string, len(mp))
	var i int
	for k, _ := range mp {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	val := make(mappySlice, len(mp)*2)
	i = 0
	for _, k := range keys {
		val[i] = k
		i++
		val[i] = mp[k]
		i++
	}
	return val
}

func (i Input) String() string {
	var buf bytes.Buffer
	codec.NewEncoder(&buf, &codec.JsonHandle{}).Encode(i)
	return buf.String()
}

var _ codec.Selfer = &OutputGroup{}

func (og OutputGroup) CodecEncodeSelf(c *codec.Encoder) {
	c.Encode(og.asMappySlice())
}

func (og *OutputGroup) CodecDecodeSelf(c *codec.Decoder) {
	// I'd love to just punt to the defaults, but the `Selfer` interface doesn't come in half.
	// SO here's a ridiculous indirection to prance around infinite recursion.
	c.MustDecode((*map[string]*Output)(og))
	// Now go back over the struct and fill in MountPath as needed from the map keys.
	for k, v := range *og {
		if v.MountPath == "" {
			v.MountPath = k
		}
	}
}

func (mp OutputGroup) asMappySlice() codec.MapBySlice {
	keys := make([]string, len(mp))
	var i int
	for k, _ := range mp {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	val := make(mappySlice, len(mp)*2)
	i = 0
	for _, k := range keys {
		val[i] = k
		i++
		val[i] = mp[k]
		i++
	}
	return val
}

func (o Output) String() string {
	var buf bytes.Buffer
	codec.NewEncoder(&buf, &codec.JsonHandle{}).Encode(o)
	return buf.String()
}
