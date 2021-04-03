package main

import (
	"strconv"
	"strings"
	"time"
)

type jsonTime time.Time

//nolint:unparam
func (t jsonTime) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(time.Time(t).Unix(), 10)), nil
}

func (t *jsonTime) UnmarshalJSON(s []byte) (err error) {
	r := strings.ReplaceAll(string(s), `"`, ``)

	q, err := strconv.ParseInt(r, 10, 64)
	if err != nil {
		return err
	}
	*(*time.Time)(t) = time.Unix(q, 0)
	return
}

func (t jsonTime) String() string { return time.Time(t).String() }
