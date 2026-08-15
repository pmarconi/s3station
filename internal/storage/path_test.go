package storage

import "testing"

func TestNormalizePrefix(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"", "", true},
		{"/", "", true},
		{"photos", "photos/", true},
		{"photos/", "photos/", true},
		{"/photos/italy/", "photos/italy/", true},
		{"photos/../secret", "", false},
		{"..", "", false},
	}
	for _, tc := range cases {
		got, err := NormalizePrefix(tc.in)
		if tc.ok && err != nil {
			t.Fatalf("NormalizePrefix(%q) err=%v", tc.in, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("NormalizePrefix(%q) expected error", tc.in)
		}
		if tc.ok && got != tc.want {
			t.Fatalf("NormalizePrefix(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	got, err := SanitizeName("  holiday.png  ")
	if err != nil || got != "holiday.png" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := SanitizeName("../etc/passwd"); err != nil {
		// filepath.Base strips the parent, leaving passwd — still a name, not a traversal
		t.Fatalf("unexpected err: %v", err)
	}
	if _, err := SanitizeName(".."); err == nil {
		t.Fatal("expected error for ..")
	}
}

func TestTrashKeys(t *testing.T) {
	key := NewTrashKey("photos/cat.jpg")
	if !IsTrashKey(key) {
		t.Fatalf("expected trash key: %s", key)
	}
	batch, original, ok := ParseTrashKey(key)
	if !ok || original != "photos/cat.jpg" || batch == "" {
		t.Fatalf("parse %q -> %q %q %v", key, batch, original, ok)
	}
	if err := GuardUserKey(".station-trash/x/y"); err == nil {
		t.Fatal("expected reserved error")
	}
	if !IsReservedName(".station-thumbs") {
		t.Fatal("thumbs prefix should be reserved")
	}
}
