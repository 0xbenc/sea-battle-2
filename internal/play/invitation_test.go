package play

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInvitationSaveReplacesPrivateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invitation.txt")
	if err := os.WriteFile(path, []byte("old invitation with a long suffix"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeInvitation(path, "new"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "new\n" {
		t.Fatalf("replacement: %q %v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("permissions: %o", info.Mode().Perm())
	}
	target := filepath.Join(dir, "untouched.txt")
	if err := os.WriteFile(target, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if err := writeInvitation(path, "latest"); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(target)
	if err != nil || string(data) != "untouched" {
		t.Fatal("save modified symlink target")
	}
	data, err = os.ReadFile(path)
	if err != nil || string(data) != "latest\n" {
		t.Fatal("save failed to replace link")
	}
}
