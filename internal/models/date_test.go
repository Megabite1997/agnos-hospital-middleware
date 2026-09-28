package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDate(t *testing.T) {
	d, err := ParseDate(" 1990-09-01 ")
	require.NoError(t, err)
	assert.Equal(t, NewDate(1990, time.September, 1), d)

	_, err = ParseDate("01/09/1990")
	assert.ErrorContains(t, err, "expected YYYY-MM-DD")
}

func TestDate_JSONRoundTrip(t *testing.T) {
	d := NewDate(1985, time.April, 12)
	b, err := json.Marshal(d)
	require.NoError(t, err)
	assert.JSONEq(t, `"1985-04-12"`, string(b))

	var got Date
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, d, got)
}

func TestDate_UnmarshalJSON_Errors(t *testing.T) {
	var d Date
	assert.Error(t, json.Unmarshal([]byte(`19900901`), &d))
	assert.Error(t, json.Unmarshal([]byte(`"1990-13-01"`), &d))
}

func TestDate_Value(t *testing.T) {
	v, err := NewDate(2000, time.January, 2).Value()
	require.NoError(t, err)
	assert.Equal(t, "2000-01-02", v)
}

func TestDate_Scan(t *testing.T) {
	var d Date
	require.NoError(t, d.Scan(time.Date(2001, 2, 3, 15, 4, 5, 0, time.FixedZone("ICT", 7*3600))))
	assert.Equal(t, NewDate(2001, time.February, 3), d)

	require.NoError(t, d.Scan("2002-03-04"))
	assert.Equal(t, NewDate(2002, time.March, 4), d)

	assert.Error(t, d.Scan("bad"))
	assert.Error(t, d.Scan(42))
}
