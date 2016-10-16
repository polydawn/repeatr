package hitch

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/go-yaml/yaml"
	"github.com/ugorji/go/codec"
	. "go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/lib/cereal"
)

var codecBounceHandler = &codec.CborHandle{}

/*
	Decodes a yaml/json stream into an object.
	Specifically meant for use with structs from the `def` package.

	May error with:

	  - `*hitch.ErrIO` for any errors in consuming `input`.
	  - `*hitch.ErrParsing` for any errors in parsing the raw input.
	  - `*def.ErrConfig` for semantic violations in the content.
	  - `*meep.ErrProgrammer` if the `val` param isn't serializable.
*/
func DecodeYaml(input io.Reader, val interface{}) {
	// Give up and buffer the entire thing; we're gonna flip it back and
	// forth several times anyway.
	byts, err := ioutil.ReadAll(input)
	if err != nil {
		panic(Meep(&ErrIO{}, Cause(err)))
	}

	// Turn tabs into spaces so that tabs are acceptable inputs.
	byts = cereal.Tab2space(byts)

	// Bounce the serial form into another temporary intermediate form.
	// Yes.  Feel the sadness in your soul.
	// This lets us feed a byte area to ugorji codec that it understands,
	//  because it doesn't have any mechanisms to accept in-memory structs.
	var raw interface{}
	if err := yaml.Unmarshal(byts, &raw); err != nil {
		panic(Meep(&ErrParsing{}, Cause(err)))
	}
	var buf bytes.Buffer
	if err := codec.NewEncoder(&buf, codecBounceHandler).Encode(raw); err != nil {
		panic(Meep(
			&ErrProgrammer{},
			Cause(Meep(
				&ErrInvalidParam{
					Param:  "val",
					Reason: fmt.Sprintf("must be serializable; encountered error %q", err),
				},
			)),
		))
	}

	// Actually decode with the smart codecs.
	if err := codec.NewDecoder(&buf, codecBounceHandler).Decode(&val); err != nil {
		panic(def.ErrConfigParsing{Msg: fmt.Sprintf("Could not parse formula: %s", err)})
	}
}
