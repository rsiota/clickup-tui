package clickup

import (
	"encoding/json"
	"strconv"
	"strings"
)

// FlexString accepts a JSON string or number.
type FlexString string

func (f *FlexString) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*f = ""
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = FlexString(s)
		return nil
	}
	*f = FlexString(strings.TrimSpace(string(b)))
	return nil
}

func (f FlexString) String() string { return string(f) }

// FlexInt64 accepts a JSON number or numeric string.
type FlexInt64 int64

func (f *FlexInt64) UnmarshalJSON(b []byte) error {
	if string(b) == "null" || string(b) == `""` {
		*f = 0
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		if s == "" {
			*f = 0
			return nil
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		*f = FlexInt64(n)
		return nil
	}
	var n int64
	if err := json.Unmarshal(b, &n); err != nil {
		var f64 float64
		if err2 := json.Unmarshal(b, &f64); err2 != nil {
			return err
		}
		*f = FlexInt64(f64)
		return nil
	}
	*f = FlexInt64(n)
	return nil
}

func (f FlexInt64) Int64() int64 { return int64(f) }
