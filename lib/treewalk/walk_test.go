package treewalk

import (
	"strings"
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

	Convey("Given a depth=2 tree", t, func() {
		root := &testNode{
			children: []*testNode{
				{},
				{},
				{},
			},
		}

		Convey("We can walk and each visitor is called once per node", func() {
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
			So(previsitCount, ShouldEqual, 4)
			So(postvisitCount, ShouldEqual, 4)
		})
	})

	Convey("Given a deep and ragged tree", t, func() {
		root := &testNode{
			value: "1",
			children: []*testNode{
				{
					value: "1.1",
					children: []*testNode{
						{value: "1.1.1"},
						{value: "1.1.2"},
						{value: "1.1.3"},
					}},
				{
					value: "1.2",
				},
				{
					value: "1.3",
					children: []*testNode{
						{
							value: "1.3.1",
							children: []*testNode{
								{value: "1.3.1.1"},
							},
						},
					},
				},
			},
		}

		Convey("We can walk and each visitor is called once per node", func() {
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
			So(previsitCount, ShouldEqual, 9)
			So(postvisitCount, ShouldEqual, 9)
		})

		Convey("Visitation occurs in order", func() {
			var record []string

			preVisit := func(n Node) error {
				record = append(record, "previsit  "+n.(*testNode).value)
				return nil
			}
			postVisit := func(n Node) error {
				record = append(record, "postvisit "+n.(*testNode).value)
				return nil
			}

			So(Walk(root, preVisit, postVisit), ShouldBeNil)
			So(strings.Join(record, "\n"), ShouldEqual, strings.Join([]string{
				"previsit  1",
				"previsit  1.1",
				"previsit  1.1.1",
				"postvisit 1.1.1",
				"previsit  1.1.2",
				"postvisit 1.1.2",
				"previsit  1.1.3",
				"postvisit 1.1.3",
				"postvisit 1.1",
				"previsit  1.2",
				"postvisit 1.2",
				"previsit  1.3",
				"previsit  1.3.1",
				"previsit  1.3.1.1",
				"postvisit 1.3.1.1",
				"postvisit 1.3.1",
				"postvisit 1.3",
				"postvisit 1",
			}, "\n"))
		})
	})
}
