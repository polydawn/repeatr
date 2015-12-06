package def

import (
	"sort"

	"github.com/ugorji/go/codec"
)

var _ codec.Selfer = &Env{}

func (e Env) CodecEncodeSelf(c *codec.Encoder) {
	c.Encode(e.asMappySlice())
}

func (mp Env) asMappySlice() codec.MapBySlice {
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

func (e *Env) CodecDecodeSelf(c *codec.Decoder) {
	c.MustDecode((*map[string]string)(e))
}
