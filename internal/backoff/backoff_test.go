package backoff

import (
	"testing"
	"time"
)

func TestDelay(t *testing.T) {
	cases := map[int]time.Duration{
		-1: 5 * time.Minute, 0: 5 * time.Minute,
		1: 5 * time.Second, 2: 15 * time.Second, 3: 60 * time.Second,
		4: 5 * time.Minute, 5: 5 * time.Minute, 1000: 5 * time.Minute,
	}
	for n, want := range cases {
		if got := Delay(n); got != want {
			t.Errorf("Delay(%d) = %v, want %v", n, got, want)
		}
	}
	if Steady != 5*time.Minute {
		t.Fatal("Steady changed")
	}
}
