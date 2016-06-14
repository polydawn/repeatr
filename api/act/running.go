package act

import (
	"polydawn.net/repeatr/api/def"
)

/*
	Evaluate a formula, yielding a runrecord.

	Incidentals like logging can be configured out of band.
*/
type FormulaRunner func(*def.Formula) *def.RunRecord
