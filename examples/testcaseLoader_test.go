package examples

import (
	"encoding/json"
	"strings"

	"github.com/warpfork/go-wish/wishfix"
)

func loadTestcase(filename string) (tc testcase) {
	tc.hunks = wishfix.MustLoadFile(filename)
	return
}

type testcase struct {
	hunks wishfix.Hunks
}

func (tc testcase) command() []string {
	return strings.Split(strings.TrimSpace(string(tc.hunks.GetSection("command"))), " ")
}
func (tc testcase) formula() []byte {
	return tc.hunks.GetSection("formula")
}
func (tc testcase) exitcode() int {
	code := 0
	_ = json.Unmarshal(tc.hunks.GetSection("exitcode"), &code)
	return code
}
func (tc testcase) stdout() []string {
	bs := tc.hunks.GetSection("stdout")
	if bs == nil {
		return nil
	}
	return strings.Split(string(bs), "\n")
}
func (tc testcase) stderr() []string {
	bs := tc.hunks.GetSection("stderr")
	if bs == nil {
		return nil
	}
	return strings.Split(string(bs), "\n")
}
