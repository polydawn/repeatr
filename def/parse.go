package def

import (
	"bytes"

	"github.com/go-yaml/yaml"
	"github.com/spacemonkeygo/errors"
	"github.com/ugorji/go/codec"

	"polydawn.net/repeatr/lib/cereal"
)

var codecBounceHandler = &codec.CborHandle{}

func ParseYaml(ser []byte) *Formula {
	// Turn tabs into spaces so that tabs are acceptable inputs.
	ser = cereal.Tab2space(ser)
	// Bounce the serial form into another temporary intermediate form.
	// Yes.  Feel the sadness in your soul.
	// This lets us feed a byte area to ugorji codec that it understands,
	//  because it doesn't have any mechanisms to accept in-memory structs.
	var raw interface{}
	if err := yaml.Unmarshal(ser, &raw); err != nil {
		panic(ConfigError.New("Could not parse formula: %s", errors.GetMessage(err)))
	}
	var buf bytes.Buffer
	if err := codec.NewEncoder(&buf, codecBounceHandler).Encode(raw); err != nil {
		panic(ConfigError.New("Could not parse formula: %s", errors.GetMessage(err)))
	}
	// Actually decode with the smart codecs.
	var frm Formula
	if err := codec.NewDecoder(&buf, codecBounceHandler).Decode(&frm); err != nil {
		panic(ConfigError.New("Could not parse formula: %s", errors.GetMessage(err)))
	}
	return &frm
}
