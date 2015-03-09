package outputs

import (
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/output"
)

func Get(desire def.Output) *output.Output {
	var output output.Output

	switch desire.Type {
	default:
		panic(def.ValidationError.New("No such output %s", desire))
	}

	return &output
}
