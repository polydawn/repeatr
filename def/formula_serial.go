package def

import (
	"sort"
	"strings"
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
	i.Filters.InitDefaults()

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
				return ConfigError.New("filter uid requires one parameter")
			}
			if words[1] == "keep" {
				f.Uid = "keep"
				break
			}
			// TODO may support special value "host" in the future to say "fuckkit, no privs" for input filters

			// for each filter:
			// - special state uninitialized
			// - special state "keep"
			// - some have special state "host"?  (only uid/gid; and probably only on input!)
			// - generic value.
			// note that not all of these values are literally useful, either.
			//  for example "host" has to be resolved to a literal, and that's from yet another
			//   layer of config resolve (because we have to keep it for reserialization at this level).
			// so that at least makes it super clear that we have a config form and
			//  that shouldn't refer to the functional form.

			// strconv.Atoi(words[1])
		case "gid":
		case "mtime":
		default:
			return ConfigError.New("unknown filter name %q", words[0])
		}
	}
	return nil
}

func (f *Filters) InitDefaults() {
	if f.Uid == "" {
		f.Uid = FilterDefaultUid
	}
	if f.Gid == "" {
		f.Gid = FilterDefaultGid
	}
	if f.Mtime == "" {
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
