package semlock

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSemaphoreLock(t *testing.T) {
	t.Run(`ensure allowed inside min max for max`, func(t *testing.T) {
		s := NewMaxSemaphoreLock(3, 10*time.Millisecond)
		spawnDistrubtor(s, 100)

		time.Sleep(5 * time.Millisecond) // wait a little so that the semaphore is disturbed

		s.IncAllowed()
		assert.Equal(t, uint64(3), s.GetAllowed())

		s.DecAllowed()
		assert.Equal(t, uint64(2), s.GetAllowed())

		s.DecAllowed()
		assert.Equal(t, uint64(1), s.GetAllowed())

		s.DecAllowed()
		assert.Equal(t, uint64(1), s.GetAllowed())

		fmt.Println(s.GetActive())
	})

	t.Run(`ensure allowed within min max for min`, func(t *testing.T) {
		s := NewMinSemaphoreLock(3, 10*time.Millisecond)
		spawnDistrubtor(s, 100)

		time.Sleep(5 * time.Millisecond) // wait a little so that the semaphore is disturbed

		s.DecAllowed()
		assert.Equal(t, uint64(1), s.GetAllowed())

		s.IncAllowed()
		assert.Equal(t, uint64(2), s.GetAllowed())

		s.IncAllowed()
		assert.Equal(t, uint64(3), s.GetAllowed())

		s.IncAllowed()
		assert.Equal(t, uint64(3), s.GetAllowed())

		fmt.Println(s.GetActive())
	})

	t.Run(`ensure active within min max for max`, func(t *testing.T) {
		s := NewMaxSemaphoreLock(3, 10*time.Millisecond)
		s.BlockUntilAllowed()
		s.ReleaseActive()
		s.BlockUntilAllowed()
		s.BlockUntilAllowed()
		s.ReleaseActive()
		s.ReleaseActive()
		s.BlockUntilAllowed()
		s.BlockUntilAllowed()
		s.BlockUntilAllowed()
		s.ReleaseActive()
		s.ReleaseActive()
		s.ReleaseActive()
	})

	t.Run(`ensure active within min max for min`, func(t *testing.T) {
		s := NewMinSemaphoreLock(3, 10*time.Millisecond)
		s.BlockUntilAllowed()
		s.ReleaseActive()
		s.IncAllowed()
		s.BlockUntilAllowed()
		s.BlockUntilAllowed()
		s.ReleaseActive()
		s.ReleaseActive()
		s.IncAllowed()
		s.BlockUntilAllowed()
		s.BlockUntilAllowed()
		s.BlockUntilAllowed()
		s.ReleaseActive()
		s.ReleaseActive()
		s.ReleaseActive()
	})

	t.Run(`random test`, func(t *testing.T) {
		s := NewMaxSemaphoreLock(3, 10*time.Millisecond)
		spawnDistrubtor(s, 100)
		for range 100 {
			s.IncOrDecAllowed(rand.Float32() < 0.5)
			time.Sleep(time.Millisecond)
		}
		fmt.Println(s.GetActive())
	})

	t.Run(`all goroutine must finish`, func(t *testing.T) {
		wg := sync.WaitGroup{}
		wg.Add(500)
		s := NewMaxSemaphoreLock(5, 1*time.Millisecond)
		count := int32(0)
		for range 500 {
			go func() {
				s.BlockUntilAllowed()
				defer s.ReleaseActive()
				time.Sleep(time.Millisecond)
				wg.Done()
				atomic.AddInt32(&count, 1)
			}()
		}
		wg.Wait()
		assert.Equal(t, int32(500), count)
	})

	t.Run(`all goroutine from channel must finish`, func(t *testing.T) {
		wg := sync.WaitGroup{}
		wg.Add(50)
		s := NewMaxSemaphoreLock(5, 1*time.Millisecond)
		count := int32(0)
		chans := make(chan int, 1000)
		go func() {
			for i := range 2000 {
				chans <- i
				time.Sleep(time.Microsecond)
			}
			close(chans)
		}()
		for range 50 { // assume 50 worker
			go func() {
				for range chans {
					s.BlockUntilAllowed()
					time.Sleep(20 * time.Microsecond)
					atomic.AddInt32(&count, 1)
					s.ReleaseActive()
				}
				wg.Done()
			}()
		}
		wg.Wait()
		assert.Equal(t, int32(2000), count)
	})
}

func spawnDistrubtor(s *SemaphoreLock, i int) {
	// spawn i goroutines to disturb the semaphore
	for range i { // make sure race condition satisfied
		go func() {
			for range 1000 {
				select {
				case <-time.After(10 * time.Millisecond):
					break
				default:
					if rand.Float32() < 0.33 {
						s.BlockUntilAllowed()
					} else {
						s.ReleaseActive()
					}
				}
			}
		}()
	}
}
