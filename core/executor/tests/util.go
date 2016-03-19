package tests

import (
	"polydawn.net/repeatr/def"
)

/*
	Return an otherwise-blank formula that has a sane rootfs.
*/
func getBaseFormula() def.Formula {
	// TODO this should have a local mirror
	return def.Formula{
		Inputs: def.InputGroup{
			"main": {
				Type:      "tar",
				MountPath: "/",
				Hash:      "aLMH4qK1EdlPDavdhErOs0BPxqO0i6lUaeRE4DuUmnNMxhHtF56gkoeSulvwWNqT",
				Warehouses: []string{
					"http+ca://repeatr.s3.amazonaws.com/assets/",
				},
			},
		},
	}
}
