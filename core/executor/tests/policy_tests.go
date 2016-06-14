package tests

import (
	"fmt"

	. "github.com/smartystreets/goconvey/convey"

	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/core/executor"
	"polydawn.net/repeatr/lib/testutil"
)

// NOTE WELL: while many of these tests embed "1000" as the low-priv UID, it is considered unspecified,
//  subject to change in the future, and it is INCORRECT for users to rely on this.

// This is not an optional feature -- EVERYONE needs to succeed at this, or else.
// AKA, the default policy MUST NOT result in uid=0.
func CheckUidBehavior(execEng executor.Executor) {
	Convey("SPEC: Process should start with appropriate UID for Policy", func(c C) {
		formula := getBaseFormula()
		formula.Action = def.Action{
			// n.b. can't use posix sh here: it doesn't initialize the uid variable.
			Entrypoint: []string{"bash", "-c", "echo :$UID:"},
		}

		Convey("The default policy should start with userland uid/gid", func() {
			soExpectSuccessAndOutput(execEng, formula, testutil.TestLogger(c),
				":1000:\n",
			)
		})

		for _, tr := range []struct {
			policy def.Policy
			uid    int
		}{
			{def.PolicyRoutine, 1000},
			{def.PolicyUidZero, 0},
			{def.PolicyGovernor, 0},
			{def.PolicySysad, 0},
		} {
			Convey(fmt.Sprintf("The %q policy should start with uid=%d", tr.policy, tr.uid), func() {
				formula.Action.Policy = tr.policy
				soExpectSuccessAndOutput(execEng, formula, testutil.TestLogger(c),
					fmt.Sprintf(":%d:\n", tr.uid),
				)
			})
		}
	})
}
