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

// don't care, use defaults... by having another anon struct.
func (fe runRecordFailureEnvelope) CodecEncodeSelf(c *codec.Encoder) {
	// Seriously, my kingdom for a less ridiculous dance here to say "DTRT".
	// We couldn't even do the inline during the already-silly custom encode
	//  func on runrecord, because we need a struct to implement `error` -.-
	// Nor could we simply *leave this method off*, because you can't do
	//  one direction only to implement the ugorji `Selfer` interface.
	// Oh, one more travesty: we ended up having to build a *custom decode*
	//  entirely, and in order to not make it spirallingly complex, I only
	//  made it support parsing where the type comes first (because otherwise
	//  I have to buffer the detail part again... in a way that I can't
	//  feed back into the codec, nor again can we spawn a new codec to handle
	//  a fresh byte stream while using the same handle and config.  Wow).
	//  WHICH MEANS.  We can't even use the anon struct; we need to go
	//  full on order override to make sure this can round-trip.
	c.MustEncode(mappySlice{
		"type", fe.Type,
		"detail", fe.Detail,
	})
}

// this method had to get WAY too fancy in order to do some basic polymorphism.
func (fe *runRecordFailureEnvelope) CodecDecodeSelf(c *codec.Decoder) {
	_, dec := codec.GenHelperDecoder(c)
	dec.ReadMapStart()
	// i need the type info before i can decode the value, but it wasn't exported in that order
	// is there an exported ability to consume the end-of-map token?! i can't see one.
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
