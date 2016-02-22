package def

import (
	"github.com/ugorji/go/codec"
)

var _assertHelper Policy
var _ codec.Selfer = &_assertHelper

func (p Policy) CodecEncodeSelf(c *codec.Encoder) {
	c.Encode(string(p))
}
func (p *Policy) CodecDecodeSelf(c *codec.Decoder) {
	var str string
	c.MustDecode(&str)
	for _, v := range PolicyValues {
		if string(v) == str {
			*p = Policy(str)
			return
		}
	}
	panic(ConfigError.New("policy value %q is not a known policy name", str))
}
