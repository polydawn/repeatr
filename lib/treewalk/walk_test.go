package treewalk

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type testNode struct {
	value string

	children []*testNode
	itrIndex int // next child offset
}

func (t *testNode) NextChild() Node {
	if t.itrIndex >= len(t.children) {
		return nil
	}
	t.itrIndex++
	return t.children[t.itrIndex-1]
}

func Test(t *testing.T) {
	Convey("Given a single node", t, func() {
		root := &testNode{}

		Convey("We can walk and each visitor is called once", func() {
			previsitCount := 0
			postvisitCount := 0

			preVisit := func(Node) error {
				previsitCount++
				return nil
			}
			postVisit := func(Node) error {
				postvisitCount++
				return nil
			}

			So(Walk(root, preVisit, postVisit), ShouldBeNil)
			So(previsitCount, ShouldEqual, 1)
			So(postvisitCount, ShouldEqual, 1)
		})
	})
}
