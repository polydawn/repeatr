package git

import (
	"fmt"
	"io"

	"github.com/src-d/go-git"
)

func ExampleBasic_printCommits() {
	r, err := git.NewRepository("https://github.com/src-d/go-git", nil)
	if err != nil {
		panic(err)
	}

	if err := r.Pull("origin", "refs/heads/master"); err != nil {
		panic(err)
	}

	iter, err := r.Commits()
	if err != nil {
		panic(err)
	}
	defer iter.Close()

	for {
		commit, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}

			panic(err)
		}

		fmt.Println(commit)
	}

	/// Output:
	// wow
}
