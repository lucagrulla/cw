package main

import (
	//"fmt"

	"github.com/stretchr/testify/assert" //"reflect"
	"io/ioutil"
	"log"
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

	s := "2d"
	x := time.Now().UTC().AddDate(0, 0, -2)

	y, m, d := x.Date()

	parsedTime, _ := timestampToTime(&s)
	assert.Equal(time.Date(y, m, d, x.Hour(), x.Minute(), 0, 0, time.UTC),
		parsedTime, "wrong parsing for input %s", s)

	s = "03d50m"
	dd, _ := time.ParseDuration(`50m`)
	x = time.Now().UTC().AddDate(0, 0, -3).Add(-dd)

	y, m, d = x.Date()

	parsedTime, _ = timestampToTime(&s)
	assert.Equal(time.Date(y, m, d, x.Hour(), x.Minute(), 0, 0, time.UTC),
		parsedTime, "wrong parsing for input %s", s)

	s = "32h"
	dd, _ = time.ParseDuration(s)
	x = time.Now().UTC().Add(-dd)

	y, m, d = x.Date()

	parsedTime, _ = timestampToTime(&s)
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

func TestCoordinatorRemoveItem(t *testing.T) {
	a := assert.New(t)
	log := log.New(ioutil.Discard, "", log.LstdFlags)

	groupTrigger1 := make(chan time.Time, 1)
	groupTrigger2 := make(chan time.Time, 1)

	channels := []chan<- time.Time{chan<- time.Time(groupTrigger1),
		chan<- time.Time(groupTrigger2)}

	coordinator := &tailCoordinator{log: log}
	coordinator.start(channels)

	coordinator.remove(channels[0])

	select {
	case _, ok := <-groupTrigger1:
		if ok {
			a.Fail("Channel should be closed.")
		}
	case <-time.After(1 * time.Second):
		a.Fail("Timeout")
	}
}
