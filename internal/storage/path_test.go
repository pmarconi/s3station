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

func TestSanitizeRelPath(t *testing.T) {
	got, err := SanitizeRelPath("vacation/raw/cat.jpg")
	if err != nil || got != "vacation/raw/cat.jpg" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := SanitizeRelPath("vacation/.station-trash/x"); err == nil {
		t.Fatal("expected reserved error")
	}
	if _, err := SanitizeRelPath("../secret"); err == nil {
		t.Fatal("expected invalid path")
	}
}

func TestPublicFolderPath(t *testing.T) {
	if PublicFolderPath("") != "/files/" {
		t.Fatal(PublicFolderPath(""))
	}
	if PublicFolderPath("photos/italy/") != "/files/photos/italy/" {
		t.Fatal(PublicFolderPath("photos/italy/"))
	}
	got, ok := PrefixFromRequestPath("/files/photos/italy/")
	if !ok || got != "photos/italy/" {
		t.Fatalf("%q %v", got, ok)
	}
	got, ok = PrefixFromRequestPath("/files/")
	if !ok || got != "" {
		t.Fatalf("root %q %v", got, ok)
	}
	if _, ok := PrefixFromRequestPath("/settings"); ok {
		t.Fatal("settings is an app route")
	}
	if _, ok := PrefixFromRequestPath("/photos/italy/"); ok {
		t.Fatal("folder paths must start with /files/")
	}
	if _, ok := PrefixFromRequestPath("/thumbs/x.webp"); ok {
		t.Fatal("thumbs is an app route")
	}
}

func TestUserRootKeys(t *testing.T) {
	if AbsUserKey(FilesPrefix, "") != "files/" {
		t.Fatal(AbsUserKey(FilesPrefix, ""))
	}
	if AbsUserKey(FilesPrefix, "photos/cat.jpg") != "files/photos/cat.jpg" {
		t.Fatal(AbsUserKey(FilesPrefix, "photos/cat.jpg"))
	}
	if AbsUserKey(FilesPrefix, "files/photos/cat.jpg") != "files/photos/cat.jpg" {
		t.Fatal("abs should be idempotent")
	}
	if RelUserKey(FilesPrefix, "files/photos/cat.jpg") != "photos/cat.jpg" {
		t.Fatal(RelUserKey(FilesPrefix, "files/photos/cat.jpg"))
	}
	if AbsUserKey(FilesPrefix, TrashPrefix+"x/a.jpg") != TrashPrefix+"x/a.jpg" {
		t.Fatal("trash keys stay at bucket root")
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
