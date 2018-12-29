package main

import (
	//"fmt"

	"github.com/stretchr/testify/assert" //"reflect"
	"testing"
	"time"
)

func TestTimestampToTime(t *testing.T) {
	assert := assert.New(t)

	a := "2017-03-12"
	parsedTime, _ := timestampToTime(&a)
	assert.Equal(time.Date(2017, 3, 12, 0, 0, 0, 0, time.UTC), parsedTime,
		"wrong parsing for input %s", a)

	a = "2017-03-12T18"
	parsedTime, _ = timestampToTime(&a)

	assert.Equal(time.Date(2017, 3, 12, 18, 0, 0, 0, time.UTC), parsedTime,
		"wrong parsing for input %s", a)

	a = "2017-03-12T18:22"
	parsedTime, _ = timestampToTime(&a)

	assert.Equal(time.Date(2017, 3, 12, 18, 22, 0, 0, time.UTC), parsedTime,
		"wrong parsing for input %s", a)

	a = "2017-03-12T18:22:23"
	parsedTime, _ = timestampToTime(&a)

	assert.Equal(time.Date(2017, 3, 12, 18, 22, 23, 0, time.UTC), parsedTime,
		"wrong parsing for input %s", a)

	a = "18"
	y, m, d := time.Now().Date()
	parsedTime, _ = timestampToTime(&a)

	assert.Equal(time.Date(y, m, d, 18, 0, 0, 0, time.UTC), parsedTime,
		"wrong parsing for input %s", a)

	a = "18:31"
	y, m, d = time.Now().Date()
	parsedTime, _ = timestampToTime(&a)

	assert.Equal(time.Date(y, m, d, 18, 31, 0, 0, time.UTC), parsedTime,
		"wrong parsing for input %s", a)
}

func TestHumanReadableTimeToTime(t *testing.T) {
	assert := assert.New(t)

	s := "32h"
	dd, _ := time.ParseDuration(s)
	x := time.Now().UTC().Add(-dd)

	y, m, d := x.Date()

	parsedTime, _ := timestampToTime(&s)
	assert.Equal(time.Date(y, m, d, x.Hour(), x.Minute(), 0, 0, time.UTC), parsedTime, "wrong parsing for input %s", s)

	s = "50m"
	dd, _ = time.ParseDuration(s)
	x = time.Now().UTC().Add(-dd)

	y, m, d = x.Date()

	parsedTime, _ = timestampToTime(&s)
	assert.Equal(time.Date(y, m, d, x.Hour(), x.Minute(), 0, 0, time.UTC), parsedTime, "wrong parsing for input %s", s)

	s = "2h30m"
	dd, _ = time.ParseDuration(s)
	x = time.Now().UTC().Add(-dd)

	y, m, d = x.Date()

	parsedTime, _ = timestampToTime(&s)
	assert.Equal(time.Date(y, m, d, x.Hour(), x.Minute(), 0, 0, time.UTC), parsedTime, "wrong parsing for input %s", s)
}

func TestWrongFormat(t *testing.T) {
	assert := assert.New(t)
	a := "log-group"
	_, err := timestampToTime(&a)
	assert.Error(err)
}
