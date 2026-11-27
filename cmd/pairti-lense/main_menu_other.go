//go:build !darwin

package main

func bindMenuItems([]menuCommand) (func(), error) { return func() {}, nil }
