package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ncruces/zenity"
)

type fileFilter struct {
	Name     string
	Patterns []string
}

func (a *App) dialogOptions(title string, filters []fileFilter) []zenity.Option {
	opts := []zenity.Option{zenity.Title(title)}
	a.mu.Lock()
	hwnd := a.hwnd
	a.mu.Unlock()
	opts = append(opts, dialogOwner(hwnd)...)
	if len(filters) > 0 {
		ff := zenity.FileFilters{}
		for _, f := range filters {
			ff = append(ff, zenity.FileFilter{Name: f.Name, Patterns: f.Patterns, CaseFold: true})
		}
		opts = append(opts, ff)
	}
	return opts
}

func dialogErr(path string, err error) (string, error) {
	if errors.Is(err, zenity.ErrCanceled) {
		return "", nil
	}
	if err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			return "", errors.New("a fájlválasztó ablak ezen a gépen nem érhető el")
		}
		return "", err
	}
	return path, nil
}

func existingDir(dir string) string {
	for dir != "" {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return documentsDir()
}

func (a *App) pickFile(title, dir string, filters []fileFilter) (string, error) {
	opts := a.dialogOptions(title, filters)
	opts = append(opts, zenity.Filename(existingDir(dir)+string(os.PathSeparator)))
	return dialogErr(zenity.SelectFile(opts...))
}

func (a *App) pickSave(title, dir, name string, filters []fileFilter) (string, error) {
	opts := a.dialogOptions(title, filters)
	opts = append(opts, zenity.Filename(filepath.Join(existingDir(dir), name)), zenity.ConfirmOverwrite())
	return dialogErr(zenity.SelectFileSave(opts...))
}

func (a *App) pickFolder(title, dir string) (string, error) {
	opts := a.dialogOptions(title, nil)
	opts = append(opts, zenity.Directory(), zenity.Filename(existingDir(dir)+string(os.PathSeparator)))
	return dialogErr(zenity.SelectFile(opts...))
}

func defaultOutputDir() string {
	return filepath.Join(documentsDir(), "Energofish hírlevelek")
}

func (a *App) pickFiles(title, dir string, filters []fileFilter) ([]string, error) {
	opts := a.dialogOptions(title, filters)
	opts = append(opts, zenity.Filename(existingDir(dir)+string(os.PathSeparator)))
	paths, err := zenity.SelectFileMultiple(opts...)
	if _, err := dialogErr("", err); err != nil {
		return nil, err
	}
	return paths, nil
}
