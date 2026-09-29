//go:build !windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/ncruces/zenity"
)

func dialogOwner(hwnd uintptr) []zenity.Option { return nil }

func documentsDir() string {
	home, _ := os.UserHomeDir()
	for _, name := range []string{"Documents", "Dokumentumok"} {
		d := filepath.Join(home, name)
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			return d
		}
	}
	return home
}

func opener() string {
	if runtime.GOOS == "darwin" {
		return "open"
	}
	return "xdg-open"
}

func openPath(p string) error { return exec.Command(opener(), p).Start() }

func openURL(u string) error { return exec.Command(opener(), u).Start() }

// runWindow: saját ablak csak Windowson van, máshol a böngésző nyílik meg.
func runWindow(a *App, url string, debug bool) bool { return false }

func showError(title, msg string) { _, _ = os.Stderr.WriteString(title + ": " + msg + "\n") }
