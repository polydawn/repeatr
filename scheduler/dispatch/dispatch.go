package schedulerdispatch

import (
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/scheduler"
	"polydawn.net/repeatr/scheduler/group"
	"polydawn.net/repeatr/scheduler/linear"
)

func Get(desire string) scheduler.Scheduler {
	var scheduler scheduler.Scheduler

	switch desire {
	case "group":
		scheduler = &group.Scheduler{}
	case "linear":
		scheduler = &linear.Scheduler{}
	default:
		panic(def.ValidationError.New("No such scheduler %s", desire))
	}

	return scheduler
}
