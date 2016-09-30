package remote

import "io"

type FormulaRunnerClient struct {
	remote io.Reader
}
