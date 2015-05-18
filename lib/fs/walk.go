package fs

import (
	"os"
	"path/filepath"
	"strings"

	"polydawn.net/repeatr/lib/treewalk"
)

type WalkFunc func(filenode *FilewalkNode) error

/*
	Walks a filesystem.

	This is much like the standard library's `path/filepath.Walk`, except
	it's based on `treewalk`, which means it supports both pre- and post-order
	traversals.

	All paths begin in `./`, and directory names are slash-suffixed.
	E.g. you'll see a series like `{"./", "./a/", "./a/b"}`, etc.
	This matches the behaviors described by `Normalize` in the `lib/fshash`.

	Symlinks are not followed.

	The traversal order of siblings is *not* guaranteed, and is *not* necessarily
	stable.

	Caveat: calling `node.NextChild()` during your walk results in undefined behavior.
*/
func Walk(basePath string, preVisit WalkFunc, postVisit WalkFunc) error {
	return treewalk.Walk(
		newFileWalkNode(basePath, "./"),
		func(node treewalk.Node) error {
			filenode := node.(*FilewalkNode)
			if preVisit != nil {
				if err := preVisit(filenode); err != nil {
					return err
				}
			}
			return filenode.prepareChildren(basePath)
		},
		func(node treewalk.Node) error {
			filenode := node.(*FilewalkNode)
			var err error
			if postVisit != nil {
				err = postVisit(filenode)
			}
			filenode.forgetChildren()
			return err
		},
	)
}

var _ treewalk.Node = &FilewalkNode{}

type FilewalkNode struct {
	Path string // relative to start
	Info os.FileInfo
	Err  error

	children []*FilewalkNode // note we didn't sort this
	itrIndex int             // next child offset
}

func (t *FilewalkNode) NextChild() treewalk.Node {
	if t.itrIndex >= len(t.children) {
		return nil
	}
	t.itrIndex++
	return t.children[t.itrIndex-1]
}

func newFileWalkNode(basePath, path string) (filenode *FilewalkNode) {
	filenode = &FilewalkNode{Path: path}
	filenode.Info, filenode.Err = os.Lstat(filepath.Join(basePath, path))
	if filenode.Err == nil && filenode.Info.IsDir() {
		if !strings.HasSuffix(filenode.Path, "/") {
			filenode.Path += "/"
		}
	}
	// don't expand the children until the previsit function
	// we don't want them all crashing into memory at once
	return
}

/*
	Expand next subtree.  Used in the pre-order visit step so we don't walk
	every dir up front.  `Walk()` wraps the user-defined pre-visit function
	to do this at the end.
*/
func (t *FilewalkNode) prepareChildren(basePath string) error {
	if !t.Info.IsDir() {
		return nil
	}
	f, err := os.Open(filepath.Join(basePath, t.Path))
	if err != nil {
		return err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return err
	}
	t.children = make([]*FilewalkNode, len(names))
	for i, name := range names {
		t.children[i] = newFileWalkNode(basePath, "./"+filepath.Join(t.Path, name))
	}
	return nil
}

/*
	Used in the post-order visit step so we don't continuously consume more
	memory as we walk.  `Walk()` wraps the user-defined post-visit function
	to do this at the end.
*/
func (t *FilewalkNode) forgetChildren() {
	t.children = nil
}
