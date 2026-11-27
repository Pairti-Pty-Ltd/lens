//go:build !darwin

package main

func installFinderOpen(func(string)) error { return nil }
func confirmFileClose(string) closeChoice  { return cancelClose }
