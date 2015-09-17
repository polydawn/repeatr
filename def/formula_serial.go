package def

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

func (f *Formula) Unmarshal(ser interface{}) error {
	mp, ok := ser.(map[string]interface{})
	if !ok {
		return newConfigValTypeError("formula", "structure")
	}

	{
		val, ok := mp["inputs"]
		if !ok {
			return newConfigValTypeError("inputs", "map")
		}
		val2, ok := val.(map[string]interface{})
		if !ok {
			return newConfigValTypeError("inputs", "map")
		}
		// though the serial representation is a map,
		//  we flip this to a slice and hold it as sorted.
		f.Inputs = make([]Input, len(val2))
		var i int
		for k, v := range val2 {
			f.Inputs[i].Name = k
			if err := f.Inputs[i].Unmarshal(v); err != nil {
				return err
			}
			i++
		}
		sort.Sort(InputsByName(f.Inputs))
	}

	{
		val, ok := mp["action"]
		if !ok {
			return newConfigValTypeError("action", "map")
		}
		if err := f.Action.Unmarshal(val); err != nil {
			return err
		}
	}

	{
		val, ok := mp["outputs"]
		if !ok {
			return newConfigValTypeError("outputs", "map")
		}
		val2, ok := val.(map[string]interface{})
		if !ok {
			return newConfigValTypeError("outputs", "map")
		}
		// though the serial representation is a map,
		//  we flip this to a slice and hold it as sorted.
		f.Outputs = make([]Output, len(val2))
		var i int
		for k, v := range val2 {
			f.Outputs[i].Name = k
			if err := f.Outputs[i].Unmarshal(v); err != nil {
				return err
			}
			i++
		}
		sort.Sort(OutputsByName(f.Outputs))
	}

	return nil
}

func (i *Input) Unmarshal(ser interface{}) error {
	// special: expect `Name` to have been set by caller.

	mp, ok := ser.(map[string]interface{})
	if !ok {
		return newConfigValTypeError("input", "structure")
	}

	val, ok := mp["type"]
	if !ok {
		return newConfigValTypeError("type", "string")
	}
	i.Type, ok = val.(string)
	if !ok {
		return newConfigValTypeError("type", "string")
	}

	val, ok = mp["hash"]
	if !ok {
		return newConfigValTypeError("hash", "string")
	}
	i.Hash, ok = val.(string)
	if !ok {
		return newConfigValTypeError("hash", "string")
	}

	val, ok = mp["mount"]
	if ok {
		i.MountPath, ok = val.(string)
		if !ok {
			return newConfigValTypeError("mount", "string")
		}
	} else {
		i.MountPath = i.Name
	}

	val, ok = mp["silo"]
	if ok {
		// TODO : we want to switch the structure here to a slice
		//	switch val2 := val.(type) {
		//	case []string:
		//		i.URI = val2
		//	case string:
		//		i.URI = []string{val2}
		//	default:
		//		return newConfigValTypeError("silo", "string or list of strings")
		//	}

		i.URI, ok = val.(string)
		if !ok {
			return newConfigValTypeError("silo", "string")
		}
	}

	return nil
}

func (a *Action) Unmarshal(ser interface{}) error {
	mp, ok := ser.(map[string]interface{})
	if !ok {
		return newConfigValTypeError("action", "map")
	}

	val, ok := mp["command"]
	if !ok {
		return newConfigValTypeError("command", "list of strings")
	}
	a.Entrypoint = coerceStringList(val)
	if a.Entrypoint == nil {
		return newConfigValTypeError("command", "list of strings")
	}

	val, ok = mp["cwd"]
	if ok {
		a.Cwd, ok = val.(string)
		if !ok {
			return newConfigValTypeError("type", "string")
		}
	}

	val, ok = mp["env"]
	if ok {
		a.Env = coerceMapStringString(val)
		if a.Env == nil {
			return newConfigValTypeError("env", "map of string->string")
		}
	}

	return nil
}

func (i *Output) Unmarshal(ser interface{}) error {
	// special: expect `Name` to have been set by caller.

	mp, ok := ser.(map[string]interface{})
	if !ok {
		return newConfigValTypeError("output", "structure")
	}

	val, ok := mp["type"]
	if !ok {
		return newConfigValTypeError("type", "string")
	}
	i.Type, ok = val.(string)
	if !ok {
		return newConfigValTypeError("type", "string")
	}

	val, ok = mp["mount"]
	if ok {
		i.MountPath, ok = val.(string)
		if !ok {
			return newConfigValTypeError("mount", "string")
		}
	} else {
		i.MountPath = i.Name
	}

	val, ok = mp["silo"]
	if ok {
		// TODO : we want to switch the structure here to a slice
		//	switch val2 := val.(type) {
		//	case []string:
		//		i.URI = val2
		//	case string:
		//		i.URI = []string{val2}
		//	default:
		//		return newConfigValTypeError("silo", "string or list of strings")
		//	}

		i.URI, ok = val.(string)
		if !ok {
			return newConfigValTypeError("silo", "string")
		}
	}

	val, ok = mp["filters"]
	if ok {
		if err := i.Filters.Unmarshal(val); err != nil {
			return err
		}
	}
	// if any (or all!) uninitialized values are left... that's fine, because
	// that means we wouldn't want to reserialize them as if they
	// had been specified.  one of the InitDefaults methods should
	// be called right before use.

	return nil
}

func (f *Filters) Unmarshal(ser interface{}) error {
	strs := coerceStringList(ser)
	if strs == nil {
		return newConfigValTypeError("filters", "list of strings")
	}
	for _, line := range strs {
		words := strings.Fields(line)
		if len(words) < 1 {
			continue
		}
		switch words[0] {
		case "uid":
			if len(words) != 2 {
				return ConfigError.New("uid filter requires one parameter")
			}
			if words[1] == "keep" {
				f.UidMode = FilterKeep
				break
			}
			n, err := strconv.Atoi(words[1])
			if err != nil || n < 0 {
				return ConfigError.New("uid filter parameter must be non-negative integer")
			}
			f.UidMode = FilterUse
			f.Uid = n
		case "gid":
			if len(words) != 2 {
				return ConfigError.New("gid filter requires one parameter")
			}
			if words[1] == "keep" {
				f.GidMode = FilterKeep
				break
			}
			n, err := strconv.Atoi(words[1])
			if err != nil || n < 0 {
				return ConfigError.New("gid filter parameter must be non-negative integer")
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
					return ConfigError.New("mtime filter parameter starting with '@' should be timestamp integer")
				}
				f.MtimeMode = FilterUse
				f.Mtime = time.Unix(int64(n), 0)
				break
			}
			// okay, no special rules matched: try to parse full thing as human date string.
			if len(words) < 2 {
				return ConfigError.New("mtime filter requires either RFC3339 date or unix timestamp denoted by prefix with '@")
			}
			date, err := time.Parse(time.RFC3339, strings.Join(words[1:], " "))
			if err != nil {
				return ConfigError.New("mtime filter requires either RFC3339 date or unix timestamp denoted by prefix with '@")
			}
			f.MtimeMode = FilterUse
			f.Mtime = date
		default:
			return ConfigError.New("unknown filter name %q", words[0])
		}
	}
	return nil
}

// Default filters for input are to respect everything.
func (f *Filters) InitDefaultsInput() {
	if f.UidMode == FilterUninitialized {
		f.UidMode = FilterKeep
	}
	if f.GidMode == FilterUninitialized {
		f.GidMode = FilterKeep
	}
	if f.MtimeMode == FilterUninitialized {
		f.MtimeMode = FilterKeep
	}
}

// Default filters for output are to flatten uid, gid, and mtime.
func (f *Filters) InitDefaultsOutput() {
	if f.UidMode == FilterUninitialized {
		f.UidMode = FilterUse
		f.Uid = FilterDefaultUid
	}
	if f.GidMode == FilterUninitialized {
		f.GidMode = FilterUse
		f.Gid = FilterDefaultGid
	}
	if f.MtimeMode == FilterUninitialized {
		f.MtimeMode = FilterUse
		f.Mtime = FilterDefaultMtime
	}
}

func coerceStringList(x interface{}) []string {
	y, ok := x.([]interface{})
	if !ok {
		return nil
	}
	z := make([]string, len(y))
	for i := range y {
		z[i], ok = y[i].(string)
		if !ok {
			return nil
		}
	}
	return z
}

func coerceMapStringString(x interface{}) map[string]string {
	y, ok := x.(map[string]interface{})
	if !ok {
		return nil
	}
	z := make(map[string]string, len(y))
	for k, v := range y {
		z[k], ok = v.(string)
		if !ok {
			return nil
		}
	}
	return z
}
