//go:build !windows

package main

// keepAwake csak Windowson tartja ébren a gépet (lásd keepawake_windows.go).
func keepAwake(bool) {}
