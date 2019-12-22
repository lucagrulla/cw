package cloudwatch

import (
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const ttl = 5 * time.Second
const purgeFreq = 6 * time.Second

func TestCache(t *testing.T) {
	l := log.New(ioutil.Discard, "", log.LstdFlags)

	a := assert.New(t)
	cache := createCache(ttl, purgeFreq, l)
	cache.Add("1", 1)
	a.True(cache.Has("1"))
	a.False(cache.Has("2"))
	time.Sleep((purgeFreq + (1 * time.Second)))

	a.False(cache.Has("1"))
	// a.True(cache.Has("2"), "latest item should be retained")

	a.Equal(cache.Size(), 0)
}
