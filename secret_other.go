//go:build !windows

package main

// Windowson kívül (fejlesztés, tesztek) nincs DPAPI: a fájlok csak a tulajdonos
// számára olvasható jogosultsággal kerülnek a lemezre.

var (
	protectSecret   func([]byte) ([]byte, error)
	unprotectSecret func([]byte) ([]byte, error)
)

const secretsProtected = false
