package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	h "energofish/hirlevel/internal/hirlevel"
)

// Beállítások: az import Excel pontos neve és mappája (a cikktörzs címe a ClientState-ben,
// a partnertörzs-források titkosítva külön fájlban vannak).

// DefaultImportExcel az import Excel alapértelmezett neve.
const DefaultImportExcel = "Energofish_partner_hirlevel_minta.xlsx"

// ImportSettings az Excel-import beállításai: a program csak ezt a nevű fájlt olvassa be.
type ImportSettings struct {
	ExcelName string `json:"excelName,omitempty"`
	ExcelDir  string `json:"excelDir,omitempty"`
}

// normalizeExcelName a megadott név ellenőrzése (csak fájlnév, .xlsx/.xlsm; kiterjesztés nélkül .xlsx).
func normalizeExcelName(n string) (string, error) {
	n = strings.TrimSpace(n)
	if n == "" {
		return DefaultImportExcel, nil
	}
	if strings.ContainsAny(n, `/\:*?"<>|`) {
		return "", errors.New("a fájlnév nem tartalmazhat mappát vagy tiltott karaktert (/ \\ : * ? \" < > |)")
	}
	ext := strings.ToLower(filepath.Ext(n))
	switch ext {
	case ".xlsx", ".xlsm":
	case "":
		n += ".xlsx"
	default:
		return "", fmt.Errorf("csak .xlsx vagy .xlsm munkafüzet lehet (megadva: %s)", ext)
	}
	return n, nil
}

func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return filepath.Dir(exe)
}

// importNameLocked az import Excel neve (a hívó tartja a zárat).
func (a *App) importNameLocked() string {
	if a.imp.ExcelName == "" {
		return DefaultImportExcel
	}
	return a.imp.ExcelName
}

// importDirLocked az import Excel mappája: a beállított, különben a program mappája.
func (a *App) importDirLocked() string {
	if a.imp.ExcelDir != "" {
		return a.imp.ExcelDir
	}
	return exeDir()
}

func (a *App) importPathLocked() string {
	return filepath.Join(a.importDirLocked(), a.importNameLocked())
}

// checkImportName: csak a beállított nevű Excel olvasható be.
func (a *App) checkImportName(name string) error {
	a.mu.Lock()
	want := a.importNameLocked()
	a.mu.Unlock()
	got := filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	if strings.EqualFold(got, want) {
		return nil
	}
	return fmt.Errorf("csak a(z) „%s” nevű Excel olvasható be (a kiválasztott: „%s”). A név a Beállítások › Import Excel részben módosítható", want, got)
}

type importInfo struct {
	ExcelName   string `json:"excelName"`
	ExcelDir    string `json:"excelDir"`
	DefaultName string `json:"defaultName"`
	DefaultDir  string `json:"defaultDir"`
	Path        string `json:"path"`
	Exists      bool   `json:"exists"`
	ModTime     string `json:"modTime,omitempty"`
}

func (a *App) importInfoLocked() importInfo {
	in := importInfo{ExcelName: a.importNameLocked(), ExcelDir: a.importDirLocked(), DefaultName: DefaultImportExcel,
		DefaultDir: exeDir(), Path: a.importPathLocked()}
	if st, err := os.Stat(in.Path); err == nil && !st.IsDir() {
		in.Exists, in.ModTime = true, st.ModTime().Format(time.RFC3339)
	}
	return in
}

func (a *App) settingsResponse() map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()
	return map[string]any{"import": a.importInfoLocked(), "feedURL": h.DefaultFeedURL}
}

func (a *App) apiSettings(w http.ResponseWriter, r *http.Request) (any, error) {
	return a.settingsResponse(), nil
}

// apiSettingsSave az import Excel nevének és mappájának mentése.
func (a *App) apiSettingsSave(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		ExcelName *string `json:"excelName"`
		ExcelDir  *string `json:"excelDir"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if req.ExcelName != nil {
		n, err := normalizeExcelName(*req.ExcelName)
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(n, DefaultImportExcel) {
			n = ""
		}
		a.imp.ExcelName = n
	}
	if req.ExcelDir != nil {
		d := strings.TrimSpace(*req.ExcelDir)
		if d != "" {
			if !filepath.IsAbs(d) {
				return nil, errors.New("a mappa teljes elérési útja kell (pl. C:\\Hírlevél)")
			}
			if st, err := os.Stat(d); err != nil || !st.IsDir() {
				return nil, fmt.Errorf("a mappa nem létezik: %s", d)
			}
		}
		a.imp.ExcelDir = d
	}
	a.scheduleSave()
	return map[string]any{"import": a.importInfoLocked(), "feedURL": h.DefaultFeedURL}, nil
}

func (a *App) apiSettingsFolder(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	dir := a.importDirLocked()
	a.mu.Unlock()
	path, err := a.pickFolder("Az import Excel mappája", dir)
	if err != nil {
		return nil, err
	}
	if path != "" {
		a.mu.Lock()
		a.imp.ExcelDir = path
		a.scheduleSave()
		a.mu.Unlock()
	}
	return a.settingsResponse(), nil
}

func (a *App) apiSettingsOpenFolder(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	dir := a.importDirLocked()
	a.mu.Unlock()
	return map[string]any{"ok": true}, openPath(dir)
}

// apiExcelImport a beállított nevű és helyű Excel betöltése.
func (a *App) apiExcelImport(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	path := a.importPathLocked()
	a.mu.Unlock()
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("nem található az import Excel: %s – tedd oda a fájlt, vagy állítsd be a nevét és a mappáját a Beállításokban", path)
	}
	if err := a.loadExcelPath(path, true); err != nil {
		return nil, err
	}
	return a.excelResponse()
}

// backupSettings a jelenlegi beállításfájl másolata visszaállítás előtt (beallitasok-mentes-….json).
func backupSettings(configPath string) string {
	if configPath == "" {
		return ""
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	dst := strings.TrimSuffix(configPath, ".json") + "-mentes-" + time.Now().Format("20060102-150405") + ".json"
	if os.WriteFile(dst, data, 0o600) != nil {
		return ""
	}
	return dst
}

// apiSettingsReset visszaállítja a program beállításait az alapállapotra (mintha most indulna
// először). A régi beállításfájlról másolat készül. Kérésre a partnertörzs-források (linkek),
// valamint a mentett partnerhalmazok és képviselő-fotók is törlődnek. A letöltött partnertörzs,
// a cikktörzs és a hozzáadott sablonok megmaradnak.
func (a *App) apiSettingsReset(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Sources bool `json:"sources"`
		B2B     bool `json:"b2b"`
	}
	_ = decode(r, &req)
	backup := backupSettings(a.configPath)
	a.mu.Lock()
	a.state = a.defaultState()
	a.excel, a.excelIssue = nil, nil
	a.imp = ImportSettings{}
	if req.B2B {
		a.b2bSet = B2BSettings{}
	} else {
		a.b2bSet.Loaded, a.b2bSet.Filter = nil, h.PartnerFilter{}
	}
	if a.saveTimer != nil {
		a.saveTimer.Stop()
	}
	a.mu.Unlock()
	a.saveNow()
	if req.Sources {
		if err := a.saveSources(map[string]string{}); err != nil {
			return nil, err
		}
	}
	return map[string]any{"ok": true, "backup": backup}, nil
}

// resetSettingsFile a -alaphelyzet kapcsolóhoz: a beállításfájlt félreteszi (másolatként megmarad),
// így a program alapállapotban indul.
func resetSettingsFile(configDir string) (string, error) {
	path := filepath.Join(configDir, "beallitasok.json")
	if _, err := os.Stat(path); err != nil {
		return "", nil
	}
	backup := backupSettings(path)
	if backup == "" {
		return "", errors.New("a beállítások nem menthetők félre")
	}
	return backup, os.Remove(path)
}
