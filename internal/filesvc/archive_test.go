package filesvc

import "testing"

func TestUniqueZipName(t *testing.T) {
	used := map[string]int{}
	if got := uniqueZipName(used, "cat.jpg"); got != "cat.jpg" {
		t.Fatalf("first = %q", got)
	}
	if got := uniqueZipName(used, "cat.jpg"); got != "cat (2).jpg" {
		t.Fatalf("second = %q", got)
	}
	if got := uniqueZipName(used, "album"); got != "album" {
		t.Fatalf("folder = %q", got)
	}
	if got := uniqueZipName(used, "album"); got != "album (2)" {
		t.Fatalf("folder clash = %q", got)
	}
}

func TestArchiveRelPath(t *testing.T) {
	tests := []struct {
		root, prefix, key string
		want              string
		ok                bool
	}{
		{"vacation", "photos/vacation/", "photos/vacation/cat.jpg", "vacation/cat.jpg", true},
		{"vacation", "photos/vacation/", "photos/vacation/raw/a.nef", "vacation/raw/a.nef", true},
		{"vacation", "photos/vacation/", "photos/vacation/", "", false},
		{"vacation", "photos/vacation/", "other/cat.jpg", "", false},
		{"depot", "", "cat.jpg", "depot/cat.jpg", true},
		{"depot", "", "albums/a.jpg", "depot/albums/a.jpg", true},
		{"depot", "", ".station-trash/x/a.jpg", "", false},
		{"depot", "", ".station-thumbs/a.webp", "", false},
	}
	for _, tt := range tests {
		got, ok := archiveRelPath(tt.root, tt.prefix, tt.key)
		if ok != tt.ok || got != tt.want {
			t.Fatalf("archiveRelPath(%q, %q, %q) = %q, %v; want %q, %v", tt.root, tt.prefix, tt.key, got, ok, tt.want, tt.ok)
		}
	}
}
