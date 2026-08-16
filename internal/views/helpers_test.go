package views

import (
	"testing"

	"station/internal/models"
)

func TestMoveHereDisabled(t *testing.T) {
	if !MoveHereDisabled(nil, "photos/", false) {
		t.Fatal("empty selection should disable")
	}
	if !MoveHereDisabled([]string{"photos/a.jpg"}, "photos/", false) {
		t.Fatal("already there should disable")
	}
	if MoveHereDisabled([]string{"photos/a.jpg"}, "albums/", false) {
		t.Fatal("valid dest should allow")
	}
	if !MoveHereDisabled([]string{"photos/"}, "photos/nested/", false) {
		t.Fatal("folder into self should disable")
	}
}

func TestCanEnterPicker(t *testing.T) {
	folder := models.Entry{Key: "albums/", Name: "albums", IsDir: true}
	if !CanEnterPicker([]string{"photos/a.jpg"}, folder) {
		t.Fatal("should enter unrelated folder")
	}
	if CanEnterPicker([]string{"albums/"}, folder) {
		t.Fatal("should not enter the folder being moved")
	}
	locked := models.Entry{Key: "secret/", Name: "secret", IsDir: true, Locked: true}
	if CanEnterPicker([]string{"photos/a.jpg"}, locked) {
		t.Fatal("locked folder should be closed")
	}
}
