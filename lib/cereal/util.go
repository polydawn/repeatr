package cereal

import (
	"bytes"
)

func Tab2space(x []byte) []byte {
	// okay so *I* think tabs are cool and really not that hard to deal with
	// flip into lines, replace leading tabs with spaces, flip back to bytes, cry at the loss of spilt cycles
	// fortunately it's all ascii transforms, so at least we don't have to convert to strings and back
	// unfortunately it's an expansion (yaml needs at least two spaces of indentation) so yep reallocations / large memmoves become unavoidable
	lines := bytes.Split(x, []byte{'\n'})
	buf := bytes.Buffer{}
	for i, line := range lines {
		for n := range line {
			if line[n] != '\t' {
				buf.Write(line[n:])
				break
			}
			buf.Write([]byte{' ', ' '})
		}
		if i != len(lines)-1 {
			buf.WriteByte('\n')
		}
	}
	return buf.Bytes()
}

func StringifyMapKeys(value interface{}) interface{} {
	switch value := value.(type) {
	case map[interface{}]interface{}:
		next := make(map[string]interface{}, len(value))
		for k, v := range value {
			next[k.(string)] = StringifyMapKeys(v)
		}
		return next
	case []interface{}:
		for i := 0; i < len(value); i++ {
			value[i] = StringifyMapKeys(value[i])
		}
		return value
	default:
		return value
	}
}
