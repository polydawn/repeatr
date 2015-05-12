package outputdispatch

import (
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/output"
	"polydawn.net/repeatr/output/tar"
	"polydawn.net/repeatr/output/tar2"
)

func Get(desire def.Output) output.Output {
	var output output.Output

	switch desire.Type {
	case "tar":
		output = tar2.New(desire)
	case "exec-tar":
		output = tar.New(desire)
	default:
		panic(def.ValidationError.New("No such output %s", desire))
	}

	return output
}
