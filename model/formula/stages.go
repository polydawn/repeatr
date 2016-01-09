package formula

import (
	"polydawn.net/repeatr/def"
)

// These types are all aliases for the same thing:
// we keep the types attached as a hint of how far along they are.
// (Even though they're structurally the same, their semantics change.)

type Commission def.Formula

type Stage2 def.Formula

type Stage3 def.Formula

type CommissionID string

type Stage2ID string

type Stage3ID string
