package def

import (
	"sort"
)

func (f *Formula) Unmarshal(ser interface{}) error {
	mp, ok := ser.(map[string]interface{})
	if !ok {
		return newConfigValTypeError("formula", "structure")
	}

	val, ok := mp["inputs"]
	if !ok {
		return newConfigValTypeError("inputs", "structure")
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
