package def

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (f *Formula) Unmarshal(ser interface{}) error {
	mp, ok := ser.(map[string]interface{})
	if !ok {
		return newConfigValTypeError("formula", "structure", describe(ser))
	}

	{
		val, ok := mp["inputs"]
		if !ok {
			return newConfigValTypeError("inputs", "map", "missing")
		}
		val2, ok := val.(map[string]interface{})
		if !ok {
			return newConfigValTypeError("inputs", "map", describe(val))
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
			return newConfigValTypeError("action", "map", "missing")
		}
		if err := f.Action.Unmarshal(val); err != nil {
			return err
		}
	}

	{
		val, ok := mp["outputs"]
		if ok {
			val2, ok := val.(map[string]interface{})
			if !ok {
				return newConfigValTypeError("outputs", "map", describe(val))
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
	}

	return nil
}

func (i *Input) Unmarshal(ser interface{}) error {
	// special: expect `Name` to have been set by caller.

	mp, ok := ser.(map[string]interface{})
	if !ok {
		return newConfigValTypeError("input", "structure", describe(ser))
	}

	val, ok := mp["type"]
	if !ok {
		return newConfigValTypeError("type", "string", "missing")
	}
	i.Type, ok = val.(string)
	if !ok {
		return newConfigValTypeError("type", "string", describe(val))
	}

	val, ok = mp["hash"]
	if !ok {
		return newConfigValTypeError("hash", "string", "missing")
	}
	i.Hash, ok = val.(string)
	if !ok {
		return newConfigValTypeError("hash", "string", describe(val))
	}

	val, ok = mp["mount"]
	if ok {
		i.MountPath, ok = val.(string)
		if !ok {
			return newConfigValTypeError("mount", "string", describe(val))
		}
	} else {
		i.MountPath = i.Name
	}

	val, ok = mp["silo"]
	if ok {
		switch val2 := val.(type) {
		case []interface{}:
			var err error
			i.Warehouses, err = coerceStringList(val2)
			if err != nil {
				return newConfigValTypeError("silo", "string or list of strings", err.Error())
			}
		case interface{}:
			str, ok := val2.(string)
			if !ok {
				return newConfigValTypeError("silo", "string or list of strings", describe(val))
			}
			i.Warehouses = []string{str}
		default:
			return newConfigValTypeError("silo", "string or list of strings", describe(val))
		}
	}

	return nil
}

func (a *Action) Unmarshal(ser interface{}) error {
	mp, ok := ser.(map[string]interface{})
	if !ok {
		return newConfigValTypeError("action", "map", describe(ser))
	}
	var err error

	val, ok := mp["command"]
	if !ok {
		return newConfigValTypeError("command", "list of strings", "missing")
	}
	a.Entrypoint, err = coerceStringList(val)
	if err != nil {
		return newConfigValTypeError("command", "list of strings", err.Error())
	}

	val, ok = mp["cwd"]
	if ok {
		a.Cwd, ok = val.(string)
		if !ok {
			return newConfigValTypeError("type", "string", describe(val))
		}
	}

	val, ok = mp["env"]
	if ok {
		a.Env, err = coerceMapStringString(val)
		if err != nil {
			return newConfigValTypeError("env", "map of string->string", err.Error())
		}
	}

	val, ok = mp["escapes"]
	if ok {
		if err := a.Escapes.Unmarshal(val); err != nil {
			return err
		}
	}

	return nil
}

func (i *Output) Unmarshal(ser interface{}) error {
	// special: expect `Name` to have been set by caller.

	mp, ok := ser.(map[string]interface{})
	if !ok {
		return newConfigValTypeError("output", "structure", describe(ser))
	}

	val, ok := mp["type"]
	if !ok {
		return newConfigValTypeError("type", "string", "missing")
	}
	i.Type, ok = val.(string)
	if !ok {
		return newConfigValTypeError("type", "string", describe(val))
	}

	val, ok = mp["mount"]
	if ok {
		i.MountPath, ok = val.(string)
		if !ok {
			return newConfigValTypeError("mount", "string", describe(val))
		}
	} else {
		i.MountPath = i.Name
	}

	val, ok = mp["silo"]
	if ok {
		switch val2 := val.(type) {
		case []interface{}:
			var err error
			i.Warehouses, err = coerceStringList(val2)
			if err != nil {
				return newConfigValTypeError("silo", "string or list of strings", err.Error())
			}
		case interface{}:
			str, ok := val2.(string)
			if !ok {
				return newConfigValTypeError("silo", "string or list of strings", describe(val))
			}
			i.Warehouses = []string{str}
		default:
			return newConfigValTypeError("silo", "string or list of strings", describe(val))
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
	strs, err := coerceStringList(ser)
	if strs == nil {
		return newConfigValTypeError("filters", "list of strings", err.Error())
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
				f.Mtime = time.Unix(int64(n), 0).UTC()
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

func (e *Escapes) Unmarshal(ser interface{}) error {
	mp, ok := ser.(map[string]interface{})
	if !ok {
		return newConfigValTypeError("escapes", "structure", describe(ser))
	}

	{
		val, ok := mp["mounts"]
		if ok {
			val2, ok := val.(map[string]interface{})
			if !ok {
				return newConfigValTypeError("mounts", "map", describe(val))
			}
			e.Mounts = make([]Mount, len(val2))
			var i int
			for k, v := range val2 {
				e.Mounts[i].TargetPath = k // may support names in the future like inputs
				if err := e.Mounts[i].Unmarshal(v); err != nil {
					return err
				}
				i++
			}
		}
	}

	return nil
}

func (m *Mount) Unmarshal(ser interface{}) error {
	switch val := ser.(type) {
	case string:
		m.SourcePath = val
		m.Writable = true
		return nil
	default:
		return newConfigValTypeError("mount", "string", describe(ser))
	}
}

func describe(x interface{}) string {
	return fmt.Sprintf("%T", x)
}

func coerceStringList(x interface{}) ([]string, error) {
	if w, ok := x.([]string); ok {
		return w, nil
	}
	y, ok := x.([]interface{})
	if !ok {
		return nil, fmt.Errorf(describe(x))
	}
	z := make([]string, len(y))
	for i := range y {
		z[i], ok = y[i].(string)
		if !ok {
			return nil, fmt.Errorf("%s at index %d", describe(x), i)
		}
	}
	return z, nil
}

func coerceMapStringString(x interface{}) (map[string]string, error) {
	y, ok := x.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf(describe(x))
	}
	z := make(map[string]string, len(y))
	for k, v := range y {
		z[k], ok = v.(string)
		if !ok {
			return nil, fmt.Errorf("%s at index %q", describe(x), k)
		}
	}
	return z, nil
}
