package def

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spacemonkeygo/errors"
	"github.com/ugorji/go/codec"
)

//
// InputGroup
//

var _ codec.Selfer = &InputGroup{}

func (ig InputGroup) CodecEncodeSelf(c *codec.Encoder) {
	c.Encode(ig.asMappySlice())
}

func (ig *InputGroup) CodecDecodeSelf(c *codec.Decoder) {
	// I'd love to just punt to the defaults, but the `Selfer` interface doesn't come in half.
	// SO here's a ridiculous indirection to prance around infinite recursion.
	c.MustDecode((*map[string]*Input)(ig))
	// Now go back over the struct and fill in MountPath as needed from the map keys.
	for k, v := range *ig {
		if v == nil {
			panic(ConfigError.New("input %q configuration is empty", k))
		}
		if v.MountPath == "" {
			v.MountPath = k
		}
	}
}

func (mp InputGroup) asMappySlice() codec.MapBySlice {
	keys := make([]string, len(mp))
	var i int
	for k := range mp {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	val := make(mappySlice, len(mp)*2)
	i = 0
	for _, k := range keys {
		val[i] = k
		i++
		val[i] = mp[k]
		i++
	}
	return val
}

//
// OutputGroup
//

var _ codec.Selfer = &OutputGroup{}

func (og OutputGroup) CodecEncodeSelf(c *codec.Encoder) {
	c.Encode(og.asMappySlice())
}

func (og *OutputGroup) CodecDecodeSelf(c *codec.Decoder) {
	// I'd love to just punt to the defaults, but the `Selfer` interface doesn't come in half.
	// SO here's a ridiculous indirection to prance around infinite recursion.
	c.MustDecode((*map[string]*Output)(og))
	// Now go back over the struct and fill in MountPath as needed from the map keys.
	for k, v := range *og {
		if v == nil {
			panic(ConfigError.New("output %q configuration is empty", k))
		}
		if v.MountPath == "" {
			v.MountPath = k
		}
	}
}

func (mp OutputGroup) asMappySlice() codec.MapBySlice {
	keys := make([]string, len(mp))
	var i int
	for k := range mp {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	val := make(mappySlice, len(mp)*2)
	i = 0
	for _, k := range keys {
		val[i] = k
		i++
		val[i] = mp[k]
		i++
	}
	return val
}

//
// Filters
//

var _ codec.Selfer = &Filters{}

func (f *Filters) CodecEncodeSelf(c *codec.Encoder) {
	c.Encode(f.AsStringSlice())
}

func (f *Filters) CodecDecodeSelf(c *codec.Decoder) {
	var strs []string
	c.MustDecode(&strs)
	f.FromStringSlice(strs)
}

func (f Filters) AsStringSlice() []string {
	var strs []string
	switch f.UidMode {
	case FilterUninitialized:
		break
	case FilterUse:
		strs = append(strs, fmt.Sprintf("uid %d", f.Uid))
	case FilterKeep:
		strs = append(strs, "uid keep")
	case FilterHost:
		panic(ConfigError.New("host modes not yet supported"))
	default:
		panic(errors.ProgrammerError.New("unrecognized filter case"))
	}
	switch f.GidMode {
	case FilterUninitialized:
		break
	case FilterUse:
		strs = append(strs, fmt.Sprintf("gid %d", f.Gid))
	case FilterKeep:
		strs = append(strs, "gid keep")
	case FilterHost:
		panic(ConfigError.New("host modes not yet supported"))
	default:
		panic(errors.ProgrammerError.New("unrecognized filter case"))
	}
	switch f.MtimeMode {
	case FilterUninitialized:
		break
	case FilterUse:
		strs = append(strs, fmt.Sprintf("mtime @%d", f.Mtime.Unix()))
	case FilterKeep:
		strs = append(strs, "mtime keep")
	case FilterHost:
		panic(ConfigError.New("host mode doesn't make sense for time"))
	default:
		panic(errors.ProgrammerError.New("unrecognized filter case"))
	}
	return strs
}

func (f *Filters) FromStringSlice(strs []string) {
	for _, line := range strs {
		words := strings.Fields(line)
		if len(words) < 1 {
			continue
		}
		switch words[0] {
		case "uid":
			if len(words) != 2 {
				panic(ConfigError.New("uid filter requires one parameter"))
			}
			if words[1] == "keep" {
				f.UidMode = FilterKeep
				break
			}
			n, err := strconv.Atoi(words[1])
			if err != nil || n < 0 {
				panic(ConfigError.New("uid filter parameter must be non-negative integer"))
			}
			f.UidMode = FilterUse
			f.Uid = n
		case "gid":
			if len(words) != 2 {
				panic(ConfigError.New("gid filter requires one parameter"))
			}
			if words[1] == "keep" {
				f.GidMode = FilterKeep
				break
			}
			n, err := strconv.Atoi(words[1])
			if err != nil || n < 0 {
				panic(ConfigError.New("gid filter parameter must be non-negative integer"))
			}
			f.GidMode = FilterUse
			f.Gid = n
		case "mtime":
			if len(words) == 2 && words[1] == "keep" {
				f.MtimeMode = FilterKeep
				break
			}
			// time may be either an RFC3339 string, or, a unix timestamp prefixed
			//  by '@' (similar to how the gnu 'date' command can be told to take unix timestamps).
			if len(words) == 2 && words[1][0] == '@' {
				n, err := strconv.Atoi(words[1][1:])
				if err != nil {
					panic(ConfigError.New("mtime filter parameter starting with '@' should be timestamp integer"))
				}
				f.MtimeMode = FilterUse
				f.Mtime = time.Unix(int64(n), 0).UTC()
				break
			}
			// okay, no special rules matched: try to parse full thing as human date string.
			if len(words) < 2 {
				panic(ConfigError.New("mtime filter requires either RFC3339 date or unix timestamp denoted by prefix with '@'"))
			}
			date, err := time.Parse(time.RFC3339, strings.Join(words[1:], " "))
			if err != nil {
				panic(ConfigError.New("mtime filter requires either RFC3339 date or unix timestamp denoted by prefix with '@'"))
			}
			f.MtimeMode = FilterUse
			f.Mtime = date.UTC()
		default:
			panic(ConfigError.New("unknown filter name %q", words[0]))
		}
	}
}
