// Fulcrum answers "do we have the leverage to do $thing?".
// This is a much more complicated question than "is uid == 0" in a capabilities world...
package fulcrum

import (
	"os"
	"runtime"

	"github.com/syndtr/gocapability/capability"

	"go.polydawn.net/repeatr/core/executor"
)

func Scan() *Fulcrum {
	var err error
	f := &Fulcrum{}
	f.onLinux = runtime.GOOS == "linux"
	f.ourUID = os.Getuid()
	if f.onLinux {
		f.ourCaps, err = capability.NewPid(0) // zero means self
		if err != nil {
			panic(err)
		}
	}
	return f
}

type Fulcrum struct {
	onLinux bool
	ourUID  int
	ourCaps capability.Capabilities // valid on linux; nil on mac (causing completely different logic).
}

// Whether we have enough caps to confidently access all of `$REPEATR_BASE/io/*`.
// We sum this up as "have CAP_DAC_OVERRIDE" (or, on mac, is uid==0).
func (f Fulcrum) CanShareIOCache() bool {
	if !f.onLinux {
		return f.ourUID == 0
	}
	return f.ourCaps.Get(capability.EFFECTIVE, capability.CAP_DAC_OVERRIDE)
}

// Whether we have enough caps to confidently use materialize files with ownership info.
// This is pretty literally "have CAP_CHOWN" (or, on mac, is uid==0).
func (f Fulcrum) CanMaterializeOwnership() bool {
	if !f.onLinux {
		return f.ourUID == 0
	}
	return f.ourCaps.Get(capability.EFFECTIVE, capability.CAP_CHOWN)
}

func (f Fulcrum) CanUseExecutor(e executor.Executor) bool {
	// TODO add some getters for this info to the executor interface!  either that or you're stuck doing typeswitches here...
	return true
}

// FUTURE: make this part of the placer/assembler selection process.
// Placers/assemblers is a slightly more complicated topic, because those systems are also trying to auto-detect what's right to do...
// and what it really should be doing is a negociation process:  caps_makes_available->[set]; user_config_accepts->[set].  find union.
