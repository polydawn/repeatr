package def

import (
	"fmt"

	"github.com/ugorji/go/codec"
)

func describe(x interface{}) string {
	return fmt.Sprintf("%T", x)
}

var _ codec.MapBySlice = mappySlice{}

type mappySlice []interface{}

func (mappySlice) MapBySlice() {}

// you can't really genericize this method, sadly:
// casting to `map[string]interface{}` implies changing the size in memory,
// and as a result, golang won't let you.
//
//func mapToMappySlice(mp map[string]interface{}) mappySlice {
//	keys := make([]string, len(mp))
//	var i int
//	for k, _ := range mp {
//		keys[i] = k
//		i++
//	}
//	sort.Strings(keys)
//	val := make(mappySlice, len(mp)*2)
//	i = 0
//	for _, k := range keys {
//		val[i] = k
//		i++
//		val[i] = mp[k]
//		i++
//	}
//	return val
//}
