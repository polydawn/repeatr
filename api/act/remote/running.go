package remote

import "io"

type RunObserverClient struct {
	remote io.Reader
}
