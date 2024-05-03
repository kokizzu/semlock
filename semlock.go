package semlock

import (
	"sync/atomic"
	"time"
)

const RHS = 0x0000_0000_FFFF_FFFF
const LHS = 0xFFFF_FFFF_0000_0000
const ShiftLHS = 32
const OneLHS = 0x1_0000_0000

// SemaphoreLock ensure that min <= allowed <= max and active <= allowed
// can be used to limit goroutine, rather than spawn and kill goroutine that can be error-prone
// would be easier for N goroutine to just run and be blocked until allowed to run
// so if active > allowed, it would eventually block next goroutine running until active <= allowed
// have to be uint64 since atomic only allow minimum 32-bit
// and we want to ensure only 1 32-bit value modified at the same time (atomically for both uint32) without lock
type SemaphoreLock struct {
	minMax        uint64
	activeAllowed uint64
	WaitDelay     time.Duration
}

func (s *SemaphoreLock) SetMin(min uint64) {
	atomic.StoreUint64(&s.minMax, min<<ShiftLHS|(s.minMax&RHS))
}

func (s *SemaphoreLock) SetMax(max uint64) {
	atomic.StoreUint64(&s.minMax, max|(s.minMax&LHS))
}

func (s *SemaphoreLock) SetActive(active uint64) {
	atomic.StoreUint64(&s.activeAllowed, (active<<ShiftLHS)|(s.activeAllowed&RHS))
}

func (s *SemaphoreLock) SetAllowed(allowed uint64) {
	atomic.StoreUint64(&s.activeAllowed, allowed|(s.activeAllowed&LHS))
}

// NewMinSemaphoreLock create new semaphore lock that starts with maximum 1
// call IncAllowed to increase the maximum allowed calls
func NewMinSemaphoreLock(max int, waitDelay time.Duration) *SemaphoreLock {
	res := &SemaphoreLock{}
	res.SetMax(uint64(max))
	res.SetMin(1)
	res.SetActive(0)
	res.SetAllowed(1)
	res.WaitDelay = waitDelay
	return res
}

// NewMaxSemaphoreLock create new semaphore lock that starts with allowed equal to max
// call DecAllowed to increase the maximum allowed calls
func NewMaxSemaphoreLock(max int, waitDelay time.Duration) *SemaphoreLock {
	res := &SemaphoreLock{}
	res.SetMax(uint64(max))
	res.SetMin(1)
	res.SetActive(0)
	res.SetAllowed(uint64(max))
	res.WaitDelay = waitDelay
	return res
}

// BlockUntilAllowed wait until can increase active (acquire 1 lock)
func (s *SemaphoreLock) BlockUntilAllowed() {
	for {
		active, allowed, activeAllowed := s.GetActiveAllowed()
		if active < allowed {
			if atomic.CompareAndSwapUint64(&s.activeAllowed, activeAllowed, activeAllowed+OneLHS) {
				return
			}
		}
		time.Sleep(s.WaitDelay)
	}
}

// ReleaseActive release semaphore (release 1 lock)
func (s *SemaphoreLock) ReleaseActive() {
	for {
		active, _, activeAllowed := s.GetActiveAllowed()
		if active == 0 { // already at minimum, so too many ReleaseActive being called (bad logic) so just ignore it
			break
		}
		if atomic.CompareAndSwapUint64(&s.activeAllowed, activeAllowed, activeAllowed-OneLHS) {
			return
		}
		time.Sleep(time.Millisecond)
	}
}

// IncAllowed increase allowed, cannot be ever be more than max
func (s *SemaphoreLock) IncAllowed() {
	for {
		activeAllowed := atomic.LoadUint64(&s.activeAllowed)
		allowed := activeAllowed & RHS
		max := s.GetMax()
		if allowed >= max { // already max
			break
		}
		if atomic.CompareAndSwapUint64(&s.activeAllowed, activeAllowed, activeAllowed+1) {
			break
		}
		time.Sleep(time.Millisecond) // wait a moment until ok
	}
}

// DecAllowed decrease allowed, cannot be ever be less than min
func (s *SemaphoreLock) DecAllowed() {
	for {
		activeAllowed := atomic.LoadUint64(&s.activeAllowed)
		allowed := activeAllowed & RHS
		min := s.GetMin()
		if allowed <= min { // already min
			break
		}
		if atomic.CompareAndSwapUint64(&s.activeAllowed, activeAllowed, activeAllowed-1) {
			break
		}
		time.Sleep(time.Millisecond) // wait a moment and try again
	}
}

// IncOrDecAllowed same as IncAllowed and DecAllowed, but only increase when true
func (s *SemaphoreLock) IncOrDecAllowed(inc bool) {
	if inc {
		s.IncAllowed()
	} else {
		s.DecAllowed()
	}
}

func (s *SemaphoreLock) GetMin() uint64 {
	return atomic.LoadUint64(&s.minMax) >> ShiftLHS
}

func (s *SemaphoreLock) GetMax() uint64 {
	return atomic.LoadUint64(&s.minMax) & RHS
}

func (s *SemaphoreLock) GetActive() uint64 {
	return atomic.LoadUint64(&s.activeAllowed) >> ShiftLHS
}

func (s *SemaphoreLock) GetAllowed() uint64 {
	return atomic.LoadUint64(&s.activeAllowed) & RHS
}

func (s *SemaphoreLock) GetActiveAllowed() (active, allowed, activeAllowed uint64) {
	activeAllowed = atomic.LoadUint64(&s.activeAllowed)
	return activeAllowed >> ShiftLHS, activeAllowed & RHS, activeAllowed
}
