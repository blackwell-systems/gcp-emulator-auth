package trace

import "testing"

func TestNowRFC3339Nano_NotEmpty(t *testing.T) {
	got := NowRFC3339Nano()
	if got == "" {
		t.Fatal("expected non-empty timestamp")
	}
}
