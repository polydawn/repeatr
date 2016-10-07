package hitch

import (
	"os"

	. "go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
)

/*
	Loads a formula from a file.
	The format may be json or yaml.

	May panic with:

	  - `*hitch.ErrIO` for any errors in reading the file.
	  - `*hitch.ErrParsing` for any errors in parsing the raw input.
	  - `*def.ErrConfig` for semantic violations in the content.
*/
func LoadFormulaFromFile(path string) *def.Formula {
	f, err := os.Open(path)
	if err != nil {
		panic(Meep(&ErrIO{}, Cause(err)))
	}
	defer f.Close()
	frm := &def.Formula{}
	DecodeYaml(f, frm)
	return frm
}
