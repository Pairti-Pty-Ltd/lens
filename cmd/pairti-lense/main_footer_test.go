package main

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/state"

	"pairti-lense/internal/report"
)

func TestFooterShowsADownload(t *testing.T) {
	s := &shell{
		line3:    state.NewSignal("Formatted"),
		words:    state.NewSignal("863 words"),
		chars:    state.NewSignal("5,367 characters"),
		caret:    state.NewSignal(""),
		foot:     state.NewSignal(""),
		transfer: "Downloading Typst 0.15.1 · 6.0 MB / 14.0 MB",
	}
	s.writeFooter()
	want := "Formatted   ·   863 words   ·   5,367 characters   ·   Downloading Typst 0.15.1 · 6.0 MB / 14.0 MB"
	if s.foot.Get() != want {
		t.Fatalf("%q", s.foot.Get())
	}
	s.transfer = ""
	s.writeFooter()
	if strings.Contains(s.foot.Get(), "Downloading") {
		t.Fatalf("cleared footer %q", s.foot.Get())
	}
	if s.foot.Get() != "Formatted   ·   863 words   ·   5,367 characters" {
		t.Fatalf("resting footer %q", s.foot.Get())
	}
}

func TestTickCopiesTheDownload(t *testing.T) {
	report.WatchTransfer(nil)
	t.Cleanup(func() {
		report.SetTransfer("")
		report.Transfer()
	})
	s := &shell{
		line3: state.NewSignal("http://127.0.0.1:64794/mcp"),
		foot:  state.NewSignal(""),
	}
	report.SetTransfer("Downloading Typst 0.15.1")
	s.tick(nil)
	if s.foot.Get() != "http://127.0.0.1:64794/mcp   ·   Downloading Typst 0.15.1" {
		t.Fatalf("%q", s.foot.Get())
	}
	report.SetTransfer("")
	s.tick(nil)
	if s.foot.Get() != "http://127.0.0.1:64794/mcp" {
		t.Fatalf("cleared %q", s.foot.Get())
	}
}
