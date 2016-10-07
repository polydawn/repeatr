package examineCmd

import (
	"archive/tar"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"sort"
	"strings"

	"go.polydawn.net/repeatr/lib/fshash"
	"go.polydawn.net/repeatr/lib/treewalk"
	"go.polydawn.net/repeatr/rio/filter"
)

func examinePath(thePath string, stdout io.Writer) {
	// Scan the whole arena contents back into a bucket of hashes and metadata.
	//  (If warehouses exposed their Buckets, that'd be handy.  But of course, not everyone uses those, so.)
	bucket := &fshash.MemoryBucket{}
	hasherFactory := sha512.New384
	filterset := filter.FilterSet{}
	if err := fshash.FillBucket(thePath, "", bucket, filterset, hasherFactory); err != nil {
		panic(err)
	}

	// Emit TDV.  (We'll quote&escape filenames so null-terminated lines aren't necessary -- this is meant for human consumption after all.)
	// Treewalk to the rescue, again.
	preVisit := func(node treewalk.Node) error {
		record := node.(fshash.RecordIterator).Record()
		m := record.Metadata
		// compute optional values
		var freehandValues []string
		if m.Linkname != "" {
			freehandValues = append(freehandValues, fmt.Sprintf("link:%q", m.Linkname))
		}
		if m.Typeflag == tar.TypeBlock || m.Typeflag == tar.TypeChar {
			freehandValues = append(freehandValues, fmt.Sprintf("major:%d", m.Devmajor))
			freehandValues = append(freehandValues, fmt.Sprintf("minor:%d", m.Devminor))
		} else if m.Typeflag == tar.TypeReg {
			freehandValues = append(freehandValues, fmt.Sprintf("hash:%s", base64.URLEncoding.EncodeToString(record.ContentHash)))
			freehandValues = append(freehandValues, fmt.Sprintf("len:%d", m.Size))
		}
		xattrsLen := len(m.Xattrs)
		if xattrsLen > 0 {
			sorted := make([]string, 0, xattrsLen)
			for k, v := range m.Xattrs {
				sorted = append(sorted, fmt.Sprintf("%q:%q", k, v))
			}
			sort.Strings(sorted)
			freehandValues = append(freehandValues, fmt.Sprintf("xattrs:[%s]", strings.Join(sorted, ",")))
		}
		// plug and chug
		fmt.Fprintf(stdout,
			"%q\t%c\t%#o\t%d\t%d\t%s\t%s\n",
			m.Name,
			m.Typeflag,
			m.Mode&07777,
			m.Uid,
			m.Gid,
			m.ModTime.UTC(),
			strings.Join(freehandValues, ","),
		)
		return nil
	}
	if err := treewalk.Walk(bucket.Iterator(), preVisit, nil); err != nil {
		panic(err)
	}
}
