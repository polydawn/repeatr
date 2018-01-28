package main

import (
	"context"
	"io"

	. "github.com/polydawn/go-errcat"

	"go.polydawn.net/go-timeless-api/repeatr"
)

func Batch(
	ctx context.Context,
	bastingPath string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) (err error) {
	defer RequireErrorHasCategory(&err, repeatr.ErrorCategory(""))

	// Load basting & compute evaluation order.
	basting, err := loadBasting(bastingPath)
	if err != nil {
		return err
	}
	stepOrder, err := batch.OrderSteps(*basting)
	if err != nil {
		return Errorf(repeatr.ErrUsage, "structurally invalid basting: %s", err)
	}

	_ = stepOrder

	return nil
}
