package testutil

import (
	"bytes"
)

/*
	Turn `[s1,s2,s3]` into `" (s1, s2, s3)"` and `[]` into `""`.

	Why?  Because each GoConvey suite has to have a unique name.
	Sometimes we invoke a method that generates suites; if we do so
	twice (presumably with interestingly different args), and we don't
	happen to have different parents, we need unique strings.
	Whee.
*/
func AdditionalDescription(addtnlDesc ...string) string {
	n := len(addtnlDesc)
	if n == 0 {
		return ""
	}
	var buf bytes.Buffer
	buf.WriteString(" (")
	for i := 0; i < n-1; i++ {
		buf.WriteString(addtnlDesc[i])
		buf.WriteString(", ")
	}
	buf.WriteString(addtnlDesc[n-1])
	buf.WriteRune(')')
	return buf.String()
}
