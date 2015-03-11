package inputdispatch

import (
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/input/dir"
	"polydawn.net/repeatr/input/tar"
)

func Get(desire def.Input) input.Input {
	var input input.Input

	switch desire.Type {
	case "dir":
		input = dir.New(desire)
	case "tar":
		input = tar.New(desire)
	default:
		panic(def.ValidationError.New("No such input %s", desire))
	}

	return input
}
