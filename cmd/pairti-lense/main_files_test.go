package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestBinCommandQuotesThePath(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip()
	}
	bin, args, err := binCommand(`/notes/annas "coverage".md`)
	if err != nil {
		t.Fatal(err)
	}
	if bin != "osascript" || len(args) != 2 {
		t.Fatalf("%s %v", bin, args)
	}
	if !strings.Contains(args[1], `delete POSIX file`) || !strings.Contains(args[1], `\"coverage\"`) {
		t.Fatalf("%s", args[1])
	}
}
