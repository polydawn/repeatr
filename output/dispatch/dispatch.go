package outputdispatch

import (
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/output"
	"polydawn.net/repeatr/output/tar"
)

func Get(desire def.Output) output.Output {
	var output output.Output

	switch desire.Type {
	case "tar":
		output = tar.New(desire)
	default:
		panic(def.ValidationError.New("No such output %s", desire))
	}

	return output
}
