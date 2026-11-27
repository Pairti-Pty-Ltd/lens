package ui

import "testing"

func TestShellFaceIsMono(t *testing.T) {
	name, err := RegisterShellFace()
	if err != nil {
		t.Fatal(err)
	}
	if name == "" || ShellFace() != name {
		t.Fatalf("face %q / %q", name, ShellFace())
	}
	if name == "Hack Nerd Font Mono" {
		t.Fatal("nerd font was chosen")
	}
}
