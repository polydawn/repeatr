package catalog

import (
	"sort"
)

var (
	_ sort.Interface = IDs{} // by simple ID lexical order
)

type IDs []ID

func (a IDs) Len() int           { return len(a) }
func (a IDs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a IDs) Less(i, j int) bool { return a[i] < a[j] }
