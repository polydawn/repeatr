package tests

import (
	"path/filepath"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/lib/testutil"
)

/*
	Return an otherwise-blank formula that has a sane rootfs.
*/
func getBaseFormula() def.Formula {
	// find local assets.  we rely on local files bootstrapped by earlier build process steps rather than have executor tests depend on networked transmats (and thus *network*).
	// is janky.  don't know of a best practice for finding your "project dir".  we just assume everyone calling this is a test two dirs deep :I
	projPath := filepath.Dir(filepath.Dir(testutil.OriginalDir()))

	return def.Formula{
		Inputs: def.InputGroup{
			"main": {
				Type:       "tar",
				MountPath:  "/",
				Hash:       "uJRF46th6rYHt0zt_n3fcDuBfGFVPS6lzRZla5hv6iDoh5DVVzxUTMMzENfPoboL",
				Warehouses: []string{"file://" + filepath.Join(projPath, "assets/ubuntu.tar.gz")},
			},
		},
	}
}
