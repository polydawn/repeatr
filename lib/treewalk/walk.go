package treewalk

import (
	"errors"
)

type Node interface {
	NextChild() Node
}

type WalkFunc func(node Node) error

/*
	SkipNode is used as a return value from WalkFuncs to indicate that the node named in the call (and all its children) are to be skipped.
	It only makes sense to return this from the pre-visit function; it's by definition too late after the post-visit function.
*/
var SkipNode = errors.New("skip this node")

/*
	Walk recursively descends a tree,
	calling `preVisit` on each node,
	then walking children,
	then calling `postVisit` on the node.

	The pre-visit function may add children.
	The post-visition function may similarly drop references to children
	(and probably should, to reduce memory use on large trees).
*/
func Walk(node Node, preVisit WalkFunc, postVisit WalkFunc) error {
	if preVisit != nil {
		err := preVisit(node)
		if err != nil {
			if err == SkipNode {
				return nil
			}
			return err
		}
	}

	for next := node.NextChild(); next != nil; next = node.NextChild() {
		if err := Walk(next, preVisit, postVisit); err != nil {
			return err
		}
	}

	if postVisit != nil {
		err := postVisit(node)
		if err != nil {
			if err == SkipNode {
				return nil
			}
			return err
		}
	}

	return nil
}
