package inputfixtures

import (
	"testing"

	"polydawn.net/repeatr/input/dispatch"
)

func TestDirInput1(t *testing.T) {
	if err := <-inputdispatch.Get(DirInput1).Apply(""); err != nil {
		t.Fatal(err)
	}
}

func TestDirInput2(t *testing.T) {
	if err := <-inputdispatch.Get(DirInput2).Apply(""); err != nil {
		t.Fatal(err)
	}
}
