package limiter

import (
	"testing"
	"time"
)

// dummy struct for benchmarking limiter
type dummy struct{ c chan int }

// dummy method for benchmarking limiter
func (d *dummy) do() { d.c <- 1 }

func TestLimiter(t *testing.T) {
	const Count = 10
	Delta := 100 * time.Millisecond
	c := make(chan int)
	d := &dummy{c}
	t0 := time.Now()
	limiter := NewLimiter(Delta, d.do)
	for i := 0; i < Count; i++ {
		<-d.c
	}
	limiter.Stop()
	t1 := time.Now()
	dt := t1.Sub(t0)
	target := Delta * Count
	slop := target * 2 / 10
	if dt < target-slop || (!testing.Short() && dt > target+slop) {
		t.Fatalf("%d %s ticks took %s, expected [%s,%s]", Count, Delta, dt, target-slop, target+slop)
	}
	time.Sleep(2 * Delta)
	select {
	case <-d.c:
		t.Fatal("Limiter did not shut down")
	default:
		// ok
	}
}

func BenchmarckLimiter(b *testing.B) {
	c := make(chan int)
	d := dummy{c}
	limiter := NewLimiter(1, d.do)
	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		<-d.c
	}
	b.StopTimer()
	limiter.Stop()
}
