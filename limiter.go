package limiter

import (
	"errors"
	"sync"
	"time"
)

const (
	limiterAsleep = iota
	limiterActive
	limiterLocked
	limiterStopped
)

// A Limiter can continuously execute an anonymous function (specifically a method) for each
// interval of the given time duration.  The function executed and the duration can be changed on
// the fly, and a Limiter can be stopped and restarted.
type Limiter struct {
	// TODO: add delay() & delayFunc(...interface{}) interface{}
	d      time.Duration
	t      *time.Ticker
	fn     func()
	stop   chan struct{}
	locker chan struct{}
	s      int
	mu     *sync.Mutex
}

// NewLimiter, returns a new Limiter with duration d, and function fn.  The duration d must be
// greater than zero; if not, NewTicker will panic.
func NewLimiter(d time.Duration, fn func()) *Limiter {
	if d <= 0 {
		panic(errors.New("non-positive interval for time.Duration"))
	}
	l := &Limiter{
		d:      d,
		t:      time.NewTicker(d),
		fn:     fn,
		stop:   make(chan struct{}),
		locker: make(chan struct{}),
		mu:     &sync.Mutex{},
	}
	go startLimiter(l)
	return l
}

// state, sets the limiter state
func (l *Limiter) state(limiterState int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.s = limiterState
}

// lock, locks the Limiter's main loop until unlock is called.
func (l *Limiter) lock() { l.locker <- struct{}{} }

// unlock, unlocks the Limiter's main loop. unlock panics if the Limiter is not locked.
func (l *Limiter) unlock() error {
	if !l.Locked() {
		panic(errors.New("cannot unlock an unlocked Limiter"))
	}
	<-l.locker
	return nil
}

// Active returns true if the limiter is running and not locked.
func (l *Limiter) Active() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.s == limiterActive
}

// Locked returns true if the limiter is locked.
func (l *Limiter) Locked() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.s == limiterLocked
}

// Reset, resets the limiter's interval and function, and restarts a stopped limiter.  Reset
// returns an error if d (time.Duration) is non-positive or the limiter is locked.
func (l *Limiter) Reset(d time.Duration, fn func()) error {
	if d <= 0 {
		return errors.New("non-positive interval for time.Duration")
	}
	if l.Locked() {
		// prevent calls to from being blocked by a locked Limiter
		return errors.New("cannot reset a locked Limiter")
	}

	// lock the main loop and swap values
	l.lock()
	defer l.unlock()
	l.t.Stop()
	l.t = time.NewTicker(d)
	l.fn = fn

	// restart Limiter if inactive
	if !l.Active() {
		go startLimiter(l)
	}
	return nil
}

// Stop, turns the Limiter off, stopping execution of its function.  Stop will unlock a locked
// limiter.  An error is returned if the Limiter is not active.  If the request to stop is not
// acknowledged Stop panics.
func (l *Limiter) Stop() (err error) {
	// arbitrary duration for timeout
	const to = time.Second * 10

	// check limiter state before sending stop request
	if !l.Active() {
		return errors.New("limiter not active")
	}
	if l.Locked() {
		l.unlock()
	}

	// send request and wait for acknowledgment
	l.t.Stop()
	l.stop <- struct{}{}
	select {
	case <-l.stop:
	case <-time.After(to):
		panic(errors.New("stop request timed out"))
	}
	return
}

// startLimiter, starts the Limiter's main loop and runs until Stop is called. The method Lock,
// Unlock can be used to pause/unpause the loop.
func startLimiter(l *Limiter) {
	l.state(limiterActive)
	defer l.state(limiterStopped)
	for {
		select {
		case <-l.t.C:
			l.fn()
		case <-l.locker:
			l.state(limiterLocked)
			l.locker <- struct{}{}
			l.state(limiterActive)
		case <-l.stop:
			l.stop <- struct{}{}
			return
		}
	}
}
