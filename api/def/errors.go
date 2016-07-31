package def

import (
	"fmt"
)

type ErrConfig struct {
	Key         string
	Msg         string
	MustBe      string
	WasActually string
}

func (e ErrConfig) Error() string {
	return e.Msg
}

func newConfigValTypeError(key, mustBe, wasActually string) error {
	msg := fmt.Sprintf("config key %q must be a %s; was %s", key, mustBe, wasActually)
	return ErrConfig{
		Key:         key,
		Msg:         msg,
		MustBe:      mustBe,
		WasActually: wasActually,
	}
}
