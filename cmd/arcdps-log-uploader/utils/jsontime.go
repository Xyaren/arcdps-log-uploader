package utils

import (
	"strconv"
	"strings"
	"time"
)

type JSONTime time.Time

func (t JSONTime) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(time.Time(t).Unix(), 10)), nil
}

func (t *JSONTime) UnmarshalJSON(s []byte) (err error) {
	r := strings.ReplaceAll(string(s), `"`, ``)

	q, err := strconv.ParseInt(r, 10, 64)
	if err != nil {
		return err
	}
	*(*time.Time)(t) = time.Unix(q, 0)
	return
}

func (t JSONTime) String() string { return time.Time(t).String() }
