//go:build windows

package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
	"github.com/jchv/go-webview2/webviewloader"
	"github.com/ncruces/zenity"
	"golang.org/x/sys/windows"
)

func dialogOwner(hwnd uintptr) []zenity.Option {
	if hwnd == 0 {
		return nil
	}
	return []zenity.Option{zenity.Attach(hwnd)}
}

func documentsDir() string {
	if p, err := windows.KnownFolderPath(windows.FOLDERID_Documents, 0); err == nil && p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Documents")
}

// openPath mappát az Intézőben, fájlt az alapértelmezett programmal nyit meg.
func openPath(p string) error {
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return exec.Command("explorer.exe", p).Start()
	}
	return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", p).Start()
}

func openURL(u string) error {
	return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", u).Start()
}

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	procGetDpiForSystem  = user32.NewProc("GetDpiForSystem")
	procSystemParamsInfo = user32.NewProc("SystemParametersInfoW")
	procShowWindow       = user32.NewProc("ShowWindow")
)

type rect struct{ Left, Top, Right, Bottom int32 }

// windowSize a képernyőhöz és a DPI-hez igazított ablakméret.
func windowSize() (w, h, minW, minH int, maximize bool) {
	scale := 1.0
	if procGetDpiForSystem.Find() == nil {
		if dpi, _, _ := procGetDpiForSystem.Call(); dpi > 0 {
			scale = float64(dpi) / 96
		}
	}
	var r rect
	workW, workH := 1600.0, 1000.0
	if ok, _, _ := procSystemParamsInfo.Call(0x0030 /* SPI_GETWORKAREA */, 0, uintptr(unsafe.Pointer(&r)), 0); ok != 0 {
		workW, workH = float64(r.Right-r.Left), float64(r.Bottom-r.Top)
	}
	w = int(min(1560*scale, workW*0.94))
	h = int(min(980*scale, workH*0.94))
	minW = int(min(960*scale, workW))
	minH = int(min(600*scale, workH))
	maximize = workW < 1280*scale || workH < 760*scale
	return
}

// webviewDataDir a WebView2 felhasználói adatmappája: mindig a felhasználó saját,
// rendszergazdai jog nélkül írható helyén (%LOCALAPPDATA%), soha nem a program mappájában.
func webviewDataDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" || !filepath.IsAbs(base) {
		if d, err := os.UserCacheDir(); err == nil {
			base = d
		} else {
			base = os.TempDir()
		}
	}
	dir := filepath.Join(base, "EnergofishHirlevel", "WebView2")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Printf("a WebView2 adatmappa nem hozható létre (%v), ideiglenes mappa lesz", err)
		dir = filepath.Join(os.TempDir(), "EnergofishHirlevel-WebView2")
		_ = os.MkdirAll(dir, 0o700)
	}
	return dir
}

// runWindow saját ablakban (WebView2) futtatja a felületet. Ha a WebView2
// futtatókörnyezet hiányzik, false-t ad, és a program a böngészőben nyílik meg.
func runWindow(a *App, url string, debug bool) bool {
	if v, err := webviewloader.GetInstalledVersion(); err != nil || v == "" {
		log.Printf("WebView2 nem érhető el: %v", err)
		return false
	}
	runtime.LockOSThread()
	w, h, minW, minH, maximize := windowSize()
	dataDir := webviewDataDir()
	// A WebView2 a környezeti változót is figyeli (és az elsőbbséget élvez a paraméterrel szemben):
	// így az adatmappa akkor is helyes, ha a paraméter átadása elromlana.
	_ = os.Setenv("WEBVIEW2_USER_DATA_FOLDER", dataDir)
	wv := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     debug,
		AutoFocus: true,
		DataPath:  dataDir,
		WindowOptions: webview2.WindowOptions{
			Title: appTitle, Width: uint(w), Height: uint(h), IconId: 1, Center: true,
		},
	})
	if wv == nil {
		log.Printf("a WebView2 ablak nem hozható létre")
		return false
	}
	hwnd := uintptr(wv.Window())
	a.mu.Lock()
	a.hwnd = hwnd
	a.mode = "webview"
	a.mu.Unlock()
	wv.SetSize(minW, minH, webview2.HintMin)
	if maximize {
		_, _, _ = procShowWindow.Call(hwnd, 3 /* SW_MAXIMIZE */)
	}
	wv.Navigate(url)
	go func() {
		<-a.quit
		wv.Dispatch(func() { wv.Terminate() })
	}()
	wv.Run()
	return true
}

// showError hibaüzenet-ablak (MessageBox), pl. ha a program el sem tud indulni.
func showError(title, msg string) {
	t, _ := windows.UTF16PtrFromString(title)
	m, _ := windows.UTF16PtrFromString(msg)
	_, _ = windows.MessageBox(0, m, t, windows.MB_OK|windows.MB_ICONERROR)
}
