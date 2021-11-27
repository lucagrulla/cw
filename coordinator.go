package main

import (
	"container/ring"
	"log"
	"sync"
	"time"
)

type tailCoordinator struct {
	targets *ring.Ring
	sync.RWMutex
	log *log.Logger
}

func (f *tailCoordinator) start(targets []chan<- time.Time) {
	f.targets = ring.New(len(targets))
	for i := 0; i < f.targets.Len(); i++ {
		f.targets.Value = targets[i]
		f.targets = f.targets.Next()
	}

	//AWS API accepts 5 reqs/sec for account
	ticker := time.NewTicker(250 * time.Millisecond)
	go func() {
		for range ticker.C {
			if f.targets == nil {
				f.log.Println("coordinator: ring buffer is empty, exiting scheduler.")
				return
			}
			f.Lock()
			x := f.targets.Value.(chan<- time.Time)
			x <- time.Now()
			f.targets = f.targets.Next()
			f.Unlock()
		}
	}()
}

func (f *tailCoordinator) remove(c chan<- time.Time) {
	f.RLock()
	initialLen := f.targets.Len()
	f.RUnlock()

	var visited int
	f.Lock()
	defer f.Unlock()

	if f.targets.Len() == 1 {
		f.targets = ring.New(0)
		f.log.Println("coordinator: single node buffer: reset", f.targets.Len())

		return
	}

	for visited = 0; visited < f.targets.Len(); visited++ {
		if f.targets.Value == c {
			targetChan := f.targets.Value.(chan<- time.Time)

			f.targets = f.targets.Prev()
			f.targets.Unlink(1)
			close(targetChan)
			f.log.Printf("coordinator: channel found and removed at index: %d\n", visited)

			break
		}
		f.targets = f.targets.Next()
	}

	if f.targets.Len() < initialLen {
		for i := 0; i < visited; i++ {
			f.targets = f.targets.Prev()
		}
	}
}
