package web

import "testing"

func TestActionKeys(t *testing.T) {
	selected := []string{"b.jpg", "a.jpg", "", "a.jpg"}
	got := actionKeys(Signals{Selected: selected})
	if len(got) != 2 || got[0] != "a.jpg" || got[1] != "b.jpg" {
		t.Fatalf("selected keys = %v", got)
	}
	got = actionKeys(Signals{TargetKey: "only.jpg", Selected: selected})
	if len(got) != 1 || got[0] != "only.jpg" {
		t.Fatalf("target key should win, got %v", got)
	}
	if got := actionKeys(Signals{}); len(got) != 0 {
		t.Fatalf("empty = %v", got)
	}
}
