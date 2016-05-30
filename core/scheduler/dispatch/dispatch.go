package schedulerdispatch

import (
	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/core/scheduler"
	"polydawn.net/repeatr/core/scheduler/group"
	"polydawn.net/repeatr/core/scheduler/linear"
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
