package thumbs

import (
	"math"
	"testing"
)

func TestFrameTimesBounds(t *testing.T) {
	const duration = 60.0
	const n = 5
	got := frameTimes(duration, n)
	if len(got) != n {
		t.Fatalf("len = %d", len(got))
	}
	lo, hi := duration*0.04, duration*0.96
	seen := map[int]int{}
	for _, ts := range got {
		if ts < lo || ts > hi {
			t.Fatalf("timestamp %v outside (%.2f, %.2f)", ts, lo, hi)
		}
		seen[int(ts)]++
	}
	if len(seen) < 3 {
		t.Fatalf("expected spread-out times, got %v", got)
	}
}

func TestFrameTimesRandom(t *testing.T) {
	const duration = 90.0
	first := frameTimes(duration, 5)
	changed := false
	for i := 0; i < 12; i++ {
		next := frameTimes(duration, 5)
		for j := range first {
			if math.Abs(first[j]-next[j]) > 0.05 {
				changed = true
				break
			}
		}
		if changed {
			break
		}
	}
	if !changed {
		t.Fatal("expected different timestamps across regenerations")
	}
}

func TestFrameTimesUnknownDuration(t *testing.T) {
	got := frameTimes(0, 5)
	if len(got) != 5 {
		t.Fatalf("len = %d", len(got))
	}
	for _, ts := range got {
		if ts <= 0 {
			t.Fatalf("expected positive fallback, got %v", got)
		}
	}
}
