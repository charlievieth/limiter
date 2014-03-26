package limiter

import (
	"testing"
	"time"
)

// Status: incomplete
// NB: most of these tests are based on tick_test.go

// TODO:
// 	- Add tests for Bucket and Ticker
// 	- Stress test Limiter
//  - Test that the changes made within Bucket.Reset() do not cause an error
//  - Refactor - this is really ugly

func TestNewLimiter(t *testing.T) {
	const Count = 10
	Delta := 100 * time.Millisecond
	c := make(chan time.Time)
	var fn func()
	fn = func() {
		c <- time.Now()
	}
	t0 := time.Now()
	limiter := NewLimiter(Delta, fn, true)
	for i := 0; i < Count; i++ {
		<-c
	}
	limiter.Stop()
	t1 := time.Now()
	dt := t1.Sub(t0)
	target := Delta * Count
	slop := target * 2 / 10
	if dt < target-slop || (!testing.Short() && dt > target+slop) {
		t.Fatalf("%d %s ticks took %s, expected [%s,%s]", Count, Delta, dt, target-slop, target+slop)
	}
	select {
	case <-c:
		t.Fatal("NewLimiter did not stop\n")
	default:
		// ok
	}
}

func TestLimiterState(t *testing.T) {
	// NB: delay has been reduced since last test
	// what is the minimum delay value ???
	const divisor = 100
	Delta := 100 * time.Millisecond
	delay := Delta / divisor

	c := make(chan time.Time)
	var fn func()
	fn = func() {
		c <- time.Now()
	}
	lim := NewLimiter(Delta, fn, true)

	time.Sleep(delay)
	if s := lim.State(); s != Active {
		t.Fatalf("invalid state %d expected %d\n", s, Active)
	}

	if err := lim.Stop(); err != nil {
		t.Fatalf("stop returned error: %s\n", err)
	}
	time.Sleep(delay)
	if s := lim.State(); s != Stopped {
		t.Fatalf("invalid state %d expected %d\n", s, Stopped)
	}

	lim.Reset(Delta, fn, true)
	time.Sleep(delay)
	if s := lim.State(); s != Active {
		t.Fatalf("after restart, invalid state %d expected %d\n", s, Active)
	}

	if err := lim.Stop(); err != nil {
		t.Fatalf("stop returned error: %s\n", err)
	}
	time.Sleep(delay)
	if s := lim.State(); s != Stopped {
		t.Fatalf("after restart, invalid state %d expected %d\n", s, Stopped)
	}

}

func TestLimiterReset(t *testing.T) {
	// Loosely based on pkg/time's tests
	// This is really really ugly
	const (
		Count   = 10
		divisor = 10
		// multiplyer = 2
	)
	Delta0 := 100 * time.Millisecond
	Delta1 := 200 * time.Millisecond
	c := make(chan time.Time)
	var fn func()
	fn = func() {
		c <- time.Now()
	}
	lim := NewLimiter(Delta0, fn, true)

	t0 := time.Now()
	for i := 0; i < Count; i++ {
		<-c
	}
	t1 := time.Now()
	lim.Reset(Delta1, fn, true)
	t2 := time.Now()
	for i := 0; i < Count; i++ {
		<-c
	}
	t3 := time.Now()
	if err := lim.Stop(); err != nil {
		t.Fatalf("Stop returned error: %s\n", err)
	}
	dt0 := t1.Sub(t0)
	dt1 := t3.Sub(t2)
	target0 := Delta0 * Count
	target1 := Delta1 * Count
	slop0 := target0 / divisor
	slop1 := target1 / divisor

	msgFmt := "Test %d: %d %s ticks took %s, expected [%s,%s]"
	if dt0 < target0-slop0 || (!testing.Short() && dt0 > target0+slop0) {
		t.Fatalf(msgFmt, 0, Count, Delta0, dt0, target0-slop0, target0+slop0)
	}
	if dt1 < target1-slop1 || (!testing.Short() && dt1 > target1+slop1) {
		t.Fatalf(msgFmt, 1, Count, Delta1, dt1, target1-slop1, target1+slop1)
	}
}
