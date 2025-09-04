package validate

import (
	"strconv"
	"strings"
)

type ErrField struct {
	Field string `json:"field"`
	Msg   string `json:"msg"`
}

type Errs []ErrField

func (e Errs) Error() string { // error interface
	var b strings.Builder
	for i, ef := range e {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(ef.Field + ": " + ef.Msg)
	}
	return b.String()
}

// Helpers
func Required(field, value string) *ErrField {
	if strings.TrimSpace(value) == "" {
		return &ErrField{Field: field, Msg: "required"}
	}
	return nil
}

func MinInt(field string, v, min int64) *ErrField {
	if v < min {
		return &ErrField{Field: field, Msg: "must be >= " + strconv.FormatInt(min, 10)}
	}
	return nil
}
