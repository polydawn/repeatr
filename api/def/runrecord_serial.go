package def

import (
	"encoding/json"
	"fmt"
	"time"

	"go.polydawn.net/meep"

	"github.com/ugorji/go/codec"
)

var _ codec.Selfer = &_assertHelper

func (rr RunRecord) CodecEncodeSelf(c *codec.Encoder) {
	// Copy pretty much the entire struct over to an anonymous one,
	//  as a way of saying "do all the normal things you would with tags";
	//  we just want to inject an Opinion on this one field.
	//
	// To be clear: this is terrible.
	//  My kingdom for a next generation of serialization/structmapping
	//  tools which actually let me pick out the *field* without this
	//  copypasta trainwreck of misplaced ambitions.
	c.MustEncode(struct {
		HID        string      `json:"HID,omitempty"`
		UID        RunID       `json:"UID"`
		Date       time.Time   `json:"when"`
		FormulaHID string      `json:"formulaHID,omitempty"`
		Results    ResultGroup `json:"results"`
		Failure    error       `json:"failure,omitempty"`
	}{
		HID:        rr.HID,
		UID:        rr.UID,
		Date:       rr.Date,
		FormulaHID: rr.FormulaHID,
		Results:    rr.Results,
		Failure: func() error {
			if rr.Failure == nil {
				return nil
			}
			return runRecordFailureEnvelope{
				failureTypeToString(rr.Failure),
				rr.Failure,
			}
		}(),
	})
}

func (rr *RunRecord) CodecDecodeSelf(c *codec.Decoder) {
	failureEnvelope := runRecordFailureEnvelope{
		Detail: json.RawMessage{},
	}
	rusrs := struct {
		HID        string      `json:"HID,omitempty"`
		UID        RunID       `json:"UID"`
		Date       time.Time   `json:"when"`
		FormulaHID string      `json:"formulaHID,omitempty"`
		Results    ResultGroup `json:"results"`
		Failure    error       `json:"failure,omitempty"`
	}{
		Failure: failureEnvelope,
	}
	c.MustDecode(&rusrs)
	rr.HID = rusrs.HID
	rr.UID = rusrs.UID
	rr.Date = rusrs.Date
	rr.FormulaHID = rusrs.FormulaHID
	rr.Results = rusrs.Results
	if failureEnvelope.Type != "" {
		realFailure := stringToBlankFailure(failureEnvelope.Type)
		c.MustDecode(&realFailure) // which bytes?  oh right: this entire api is streaming in a way that's profoundly unhelpful right now.
		// AYLMAO if you try to put a decode method on this, this fucking lib doesn't understand the asym, so i have to manually encode it too?!
		// this couldn't be more of a shitshow of spaghetti code if someone started out with the intention of making goddamn pasta
		rr.Failure = realFailure
	}
}

type runRecordFailureEnvelope struct {
	Type   string      `json:"type"`
	Detail interface{} `json:"detail"`
}

func (runRecordFailureEnvelope) Error() string { return "" }

func failureTypeToString(e error) string {
	switch e.(type) {
	case *ErrConfigParsing:
		return "ErrConfigParsing"
	case *ErrConfigValidation:
		return "ErrConfigValidation"
	case *ErrWarehouseUnavailable:
		return "ErrWarehouseUnavailable"
	case *ErrWarehouseProblem:
		return "ErrWarehouseProblem"
	case *ErrWareDNE:
		return "ErrWareDNE"
	case *ErrWareCorrupt:
		return "ErrWareCorrupt"
	default:
		panic(meep.Meep(
			&meep.ErrProgrammer{Msg: "Internal Error not suitable for API"},
			meep.Cause(e),
		))
	}
}

func stringToBlankFailure(typ string) error {
	switch typ {
	case "ErrConfigParsing":
		return &ErrConfigParsing{}
	case "ErrConfigValidation":
		return &ErrConfigValidation{}
	case "ErrWarehouseUnavailable":
		return &ErrWarehouseUnavailable{}
	case "ErrWarehouseProblem":
		return &ErrWarehouseProblem{}
	case "ErrWareDNE":
		return &ErrWareDNE{}
	case "ErrWareCorrupt":
		return &ErrWareCorrupt{}
	default:
		panic(&ErrUnmarshalling{
			Msg: fmt.Sprintf("cannot unmarshal error type: %q is not a known type", typ),
		})
	}
}
