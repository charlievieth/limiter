package limiter

// DEVELOPMENT BRANCH

import (
	"errors"
	"sync"
	"time"
)

// Limiter state constants.
const (
<<<<<<< HEAD
<<<<<<< HEAD
	Asleep = iota
=======
	Asleep = iota // Asleep signals an uninitialized Limiter
>>>>>>> development
=======
	Asleep = iota
>>>>>>> development
	Active
	Locked
	Stopped
)

// A Limiter can continuously execute a Limiter function (specifically a
// method) for each interval of the given time duration.  The function executed
// and the duration can be changed on the fly, and a Limiter can be stopped and
// restarted.
type Limiter struct {
	// state
	s int           // state
	d time.Duration // tick interval
	t *time.Ticker  // ticker

	// function to be executed for each tick
	fn    func(*Limiter) // function called for each tick of t
	arg   interface{}    // arg for fn -> TODO: improve this comment
	async bool           // call arg in its own goroutine

	// delay
	delay *Delay

	// lock/state control
	locker chan struct{} // lock semaphore
	stop   chan struct{} // stop semaphore
	mu     *sync.Mutex   // mutual exclusion lock for thread safe state changes
}

// NewLimiter, returns a new Limiter with duration d, and function fn. If
// async is true then fn is called in its own goroutine.  The duration d
// must be greater than zero; if not, NewTicker will panic.
func NewLimiter(d time.Duration, fn func(), async bool) *Limiter {
	// NewTicker will panic if d <= 0
	tick := time.NewTicker(d)
	l := &Limiter{
		d:      d,
		t:      tick,
		arg:    fn,
		async:  async,
		stop:   make(chan struct{}),
		locker: make(chan struct{}),
		mu:     &sync.Mutex{},
	}
	if async {
		l.fn = goFunc
	} else {
		l.fn = syncFunc
	}
	go startLimiter(l)
	return l
}

// syncFunc calls l.arg synchronously.
func syncFunc(l *Limiter) {
	l.arg.(func())()
}

// goFunc calls l.arg in its own goroutine.
func goFunc(l *Limiter) {
	go l.arg.(func())()
}

// delayFunc, delays the call to l.arg by the duration returned from DelayFunc.
// The delay duration is calculated as (DelayFunc + Limiter.Duration) - the time
// since l.arg was last called.
func delayFunc(l *Limiter) {
	dur := l.delay.delfn(*l.delay)
	a := time.After(dur + l.d - time.Since(l.delay.last))
	select {
	case <-a:
		l.delay.limfn(l)
	case <-l.stop:
		l.fn = l.delay.limfn
		l.delay = &Delay{}
		return
	}
	if dur > 0 {
		l.delay.last = time.Now()
		l.delay.Prev = dur
		l.delay.Count++
	} else {
		l.fn = l.delay.limfn
		l.delay = &Delay{}
		select {
		case <-l.t.C:
			// drain one tick
		case <-l.stop:
			// prevent block
		}
	}
}

// state, sets the Limiter state.
func (l *Limiter) state(LimiterState int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.s = LimiterState
}

// State returns the Limiter's current state (Asleep, Locked, Unlocked or Stopped).
func (l *Limiter) State() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.s
}

// lock, blocks until a lock on the Limiter's main loop is acquired and blocks
// until unlock is called.  Only one function can have a lock on the limiter
// and a call to unlock should be immediately deferred following any call to
// lock.
//
// An error is returned if the stop channel is closed, this is to prevent calls
// made while the Limiter is in the process of stopping from succeeding.
func (l *Limiter) lock() (err error) {
	select {
	case l.locker <- struct{}{}:
	case <-l.stop:
		err = errors.New("limiter stopped")
	}
	return
}

<<<<<<< HEAD
// unlock, unlocks the Limiter's main loop. unlock panics if the Limiter is not locked.
<<<<<<< HEAD
func (l *Limiter) unlock() error {
=======
func (l *Limiter) unlock() {
>>>>>>> development
	if l.State() == Locked {
		panic(errors.New("cannot unlock an unlocked Limiter"))
	}
=======
// unlock, unlocks the Limiter's main loop.  Any function that acquires a lock
// should immediately defer a call to unlock.
func (l *Limiter) unlock() {
>>>>>>> development
	<-l.locker
<<<<<<< HEAD
	return nil
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
=======
>>>>>>> development
}

// Reset changes the Limters values and will restart the Limiter if stopped.
// If duration d is less than zero than an error is returned, and if stopped
// the Limiter will not be restarted (as this would cause a panic).
// func (l *Limiter) Reset(d time.Duration, fn func(), async bool) error {
func (l *Limiter) Reset(d time.Duration, fn func(), async bool) error {
	return l.reset(d, fn, async)
}

// reset, is the private reset interface for Limiter and Limiter wrappers.
// The arguments are of type interface to simplify checking for nil values.
func (l *Limiter) reset(d, fn, async interface{}) error {
	state := l.State()
	if state != Stopped {
		l.lock()
		defer l.unlock()
	}
	if t, isTime := d.(time.Duration); isTime {
		if t <= 0 {
			return errors.New("non-positive interval for duration")
		}
		l.t.Stop()
		l.d = t
		l.t = time.NewTicker(t)
	}
	if f, isFunc := fn.(func()); isFunc {
		l.arg = f
	}
	if b, isBool := async.(bool); isBool {
		if b {
			l.fn = goFunc
		} else {
			l.fn = syncFunc
		}
	}
	if state == Stopped {
		l.stop = make(chan struct{})
		go startLimiter(l)
	}
	return nil
}

<<<<<<< HEAD
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

<<<<<<< HEAD
	if l.Locked() {
=======
=======
// Stop, turns the Limiter off, stopping execution of its function.  Stop will
// unlock a locked Limiter.  An error is returned if the Limiter is not active.
// If the request to stop is not acknowledged Stop panics.
func (l *Limiter) Stop() error {
>>>>>>> development
	// check Limiter state before proceeding
	if l.State() == Stopped {
		return errors.New("Limiter already stopped")
	}
<<<<<<< HEAD
	if l.State() == Locked {
>>>>>>> development
		l.unlock()
=======
	if err := l.lock(); err != nil {
		return err
>>>>>>> development
	}
	defer l.unlock()

	l.t.Stop()
	close(l.stop)

	return nil
}

// startLimiter, starts the Limiter's main loop and runs until Stop is called.
// The methods Lock, Unlock can be used to control the loop.
func startLimiter(l *Limiter) {
	l.state(Active)
	defer l.state(Stopped)
	for {
		select {
		case <-l.t.C:
			l.fn(l)
		case <-l.locker:
			l.state(Locked)
			l.locker <- struct{}{}
			l.state(Active)
		case <-l.stop:
			return
		}
	}
}

// A DelayFunc determines the delay duration of a delayed Limiter and signals
// the end of a delay by returning a duration less than or equal to zero.
//
// The Delay argument informs the DelayFunc of the time when it was first called
// (Start), number of times it has been called (Count), and the last delay
// duration it returned (Prev).
type DelayFunc func(Delay) time.Duration

// A Delay provides relevant information to a DelayFunc.
type Delay struct {
	// Helper values provided to DelayFunc
	Start time.Time     // time when delay started
	Count int           // count of calls to DelayFunc
	Prev  time.Duration // duration last returned by

	// last is the time when Delay was last called and is used to calculate
	// the actual delay duration.
	//
	// The actual delay duration is calculated as the sum of the Limiter's
	// duration plus the duration returned from DelayFunc minus the time
	// since Delay last called the Limiter's function.
	last time.Time

	delfn DelayFunc // DelayFunc

	// When delayed, a Limiter's function is replaced by delayFunc.
	// limfn holds the Limiter's original function and is called
	// after each delay elapses.
	limfn func(*Limiter)
}

// Delay, adds duration delay to the Limiter's period (time between calls to arg)
// for duration, duration.  For example if the Lmiter's initial duration is 1s,
// delay = 500ms and duration = 15s; the Limiter's fn will be called once every
// 1500ms for 15s after which calls to fn will resume at a rate of once every
// second (1s).
//
// If the Limiter is already delayed than the last delay is replaced with the
// new delay.
//
// An error is returned if delay or duration are non-positive intervals or if
// the Limiter is stopped.
func (l *Limiter) Delay(delay, duration time.Duration) error {
	if delay <= 0 {
		return errors.New("non-positive interval for delay")
	}
	if duration <= 0 {
		return errors.New("non-positive interval for duration")
	}
	fn := func(d Delay) time.Duration {
		if time.Since(d.Start) < duration {
			return delay
		}
		return 0
	}
	return l.newDelay(fn)
}

// DelayFunc delays the Limiter's period by the duration returned from DelayFunc
// fn until the duration returned from fn is less than or equal to zero.
// An error is returned if the Limiter is stopped.
func (l *Limiter) DelayFunc(fn DelayFunc) (err error) {
	err = l.newDelay(fn)
	return
}

// newDelay, adds a limiter with DelayFunc fn to Limiter.  The Limiter will call
// delayFunc until the duration return by DelayFunc fn is less than or equal to
// zero.
func (l *Limiter) newDelay(fn DelayFunc) error {
	if l.State() == Stopped {
		return errors.New("cannot delay stopped Limiter")
	}

	// acquire lock before initializing a new Delay
	l.lock()
	defer l.unlock()

	t := time.Now()
	l.delay = &Delay{
		Start: t,
		Count: 0,
		Prev:  0,
		last:  t,
		delfn: fn,
		limfn: l.fn,
	}
	l.fn = delayFunc
	return nil
}

// A Limiter is similar to the time packages Ticker and like a Ticker it
// delivers `ticks' of a clock at intervals. Unlike a Ticker, a Limiter can be
// paused, stopped, restarted and it's duration can be changed on the fly.
type Ticker struct {
	C chan time.Time
	*Limiter
}

// NewTicker return a new Limiter containing a channel that will send the time
// with a period specified by the duration argument.  The duration d must be
// greater than zero; if not, NewTicker will panic.
func NewTicker(d time.Duration) *Ticker {
	t := &Ticker{}
	c := make(chan time.Time)
	t.C = c
	t.Limiter = NewLimiter(d, t.sendTime, true)
	return t
}

// send time on channel C, drop ticks as necessary
func (t *Ticker) sendTime() {
	select {
	case t.C <- time.Now():
	default:
		// drop tick
	}
}

// Reset the Ticker's duration, if the Ticker is stopped it will be restarted.
// An error is returned if duration d is less than or equal to zero.
func (t *Ticker) Reset(d time.Duration) error {
	return t.reset(d, nil, nil)
}

// A Bucket is a convenience wrapper for Limiter that implements a token bucket.
type Bucket struct {
	capacity int
	tokens   chan struct{}
	*Limiter
	// fill     func(*Bucket)
}

// NewBucket returns a new Bucket (token bucket) with capacity i, and a refill
// period of duration d.
// NewBucket panics if i or d are less than or equal to zero.
func NewBucket(n int, d time.Duration) *Bucket {
	if n <= 0 {
		panic(errors.New("non-positive value n for Bucket capacity"))
	}
	b := &Bucket{}
	t := make(chan struct{}, n)
	b.tokens = t
	b.capacity = n
	if n == 1 {
		b.Limiter = NewLimiter(d, b.fillOne, false)
	} else {
		b.Limiter = NewLimiter(d, b.fillMany, false)
	}
	return b
}

func (b *Bucket) fillOne() {
	select {
	case b.tokens <- struct{}{}:
	default:
	}
}

func (b *Bucket) fillMany() {
	for i := 0; i < b.capacity; i++ {
		select {
		case b.tokens <- struct{}{}:
		default:
			return
		}
	}
}

// Withdraw blocks until n tokens have been withdrawn from the Buckket.  An
// error is returned if the bucket is stopped before the withdraw completes.
func (b *Bucket) Withdraw(n int) error {
	for i := 0; i < n; i++ {
		select {
		case <-b.tokens:
		case <-b.stop:
			return errors.New("Bucket stopped before withdraw completed")
		}
	}
	return nil
}

// Reset changes the token bucket's capacity and duration values, and will
// restart a stopped token bucket.  An error is returned if either n or d are
// less than zero.
func (b *Bucket) Reset(n int, d time.Duration) error {
	// WARN: Changes here are made being made on a unlocked (live) Bucket.
	if n <= 0 {
		return errors.New("non-positive value n for Bucket capacity")
	}
	b.capacity = n
	if n == 1 {
		return b.reset(d, b.fillOne, false)
	} else {
		return b.reset(d, b.fillMany, false)
	}
}
