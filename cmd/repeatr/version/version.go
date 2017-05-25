package version

/*
	Values injected by 'ldflags' -- these vars will be the "unknown" value
	unless you use our blessed build script, which correctly determines
	and supplies values at compile time that override these placeholders.
*/
var (
	GitCommit     string = "!!unknown!!"
	GitDirty      string = "!!unknown!!"
	GitCommitDate string = "!!unknown!!"
	GitAuthorDate string = "!!unknown!!"
	GitTreeHash   string = "!!unknown!!"
)
