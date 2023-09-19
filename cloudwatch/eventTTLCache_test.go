package cloudwatch

import (
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const ttl = 1 * time.Second
const purgeFreq = 2 * time.Second

func TestCache(t *testing.T) {
	l := log.New(io.Discard, "", log.LstdFlags)

	a := assert.New(t)
	cache := createCache(ttl, purgeFreq, l)
	cache.Add("1", 1)
	cache.Add("2", 2)
	cache.Add("3", 3)

	a.True(cache.Has("1"))
	a.True(cache.Has("2"))
	a.True(cache.Has("3"))

	time.Sleep((purgeFreq + (1 * time.Second))) //wait for the cache janitor to kick in

	a.False(cache.Has("1"))
	a.False(cache.Has("2"))

	a.True(cache.Has("3"), "last added item should be retained")

	a.Equal(cache.Size(), 1)
}
