package limiter

// DEVELOPMENT BRANCH

import (
	"errors"
	"sync"
	"time"
)

const (
	Asleep = iota // Asleep signals an uninitialized Limiter
	Active
	Locked
	Stopped
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

// State returns the Limiter state (Asleep, Locked, Unlocked or Stopped).
func (l *Limiter) State() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.s
}

// lock, locks the Limiter's main loop until unlock is called.
func (l *Limiter) lock() { l.locker <- struct{}{} }

// unlock, unlocks the Limiter's main loop. unlock panics if the Limiter is not locked.
func (l *Limiter) unlock() {
	if l.State() == Locked {
		panic(errors.New("cannot unlock an unlocked Limiter"))
	}
	<-l.locker
}

// Active returns true if the limiter is running and not locked.
func (l *Limiter) Active() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.s == Active
}

// Locked returns true if the limiter is locked.
func (l *Limiter) Locked() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.s == Locked
}

func (l *Limiter) stopped() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.s == Stopped
}

// Reset, resets the limiter's interval and function, and restarts a stopped limiter.  Reset
// returns an error if d (time.Duration) is non-positive or the limiter is locked.
func (l *Limiter) Reset(d time.Duration, fn func()) error {
	if d <= 0 {
		return errors.New("non-positive interval for time.Duration")
	}
	if l.State() == Locked {
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

// Pause, will pause (lock) the Limiter for at least duration d starting from when Pause was
// called.  Pause does not check if the Limiter and must acquire a lock before returning,
// therefore if Pause
func (l *Limiter) Pause(d time.Duration) error {
	if l.State() == Stopped {
		return errors.New("cannot pause")
	}
	a := time.After(d)
	l.lock()
	defer l.unlock()
	<-a
}

// Stop, turns the Limiter off, stopping execution of its function.  Stop will unlock a locked
// limiter.  An error is returned if the Limiter is not active.  If the request to stop is not
// acknowledged Stop panics.
func (l *Limiter) Stop() (err error) {
	// arbitrary duration for timeout
	const to = time.Second * 10

	// check Limiter state before proceeding
	if l.State() == Stopped {
		return errors.New("Limiter already stopped")
	}
	if l.State() == Locked() {
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
	l.state(Active)
	defer l.state(Stopped)
	for {
		select {
		case <-l.t.C:
			l.fn()
		case <-l.locker:
			l.state(Locked)
			l.locker <- struct{}{}
			l.state(Active)
		case <-l.stop:
			l.stop <- struct{}{}
			return
		}
	}
}
