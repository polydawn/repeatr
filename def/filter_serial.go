package def

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spacemonkeygo/errors"
	"github.com/ugorji/go/codec"
)

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
