package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransferCrypto(t *testing.T) {
	plain := []byte("titkos tartalom")
	enc, err := encryptTransfer(plain, "jo-jelszo-123")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(enc, []byte(transferMagic)) || bytes.Contains(enc, plain) {
		t.Fatal("a fájl nem titkosított")
	}
	if got, err := decryptTransfer(enc, "jo-jelszo-123"); err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("visszafejtés: %v", err)
	}
	if _, err := decryptTransfer(enc, "rossz-jelszo"); err != errTransferAuth {
		t.Errorf("rossz jelszó: %v", err)
	}
	bad := append([]byte{}, enc...)
	bad[len(bad)-5] ^= 1
	if _, err := decryptTransfer(bad, "jo-jelszo-123"); err != errTransferAuth {
		t.Errorf("módosított fájl: %v", err)
	}
	head := append([]byte{}, enc...)
	head[len(transferMagic)+3] ^= 1 // a só is hitelesített
	if _, err := decryptTransfer(head, "jo-jelszo-123"); err == nil {
		t.Error("módosított fejléc")
	}
	if _, err := decryptTransfer([]byte("valami más"), "x"); err == nil {
		t.Error("idegen fájl")
	}
}

// Teljes átvitel: a „forrásgép” minden beállítása egy titkosított fájlon át a „célgépre” kerül.
func TestTransferRoundTrip(t *testing.T) {
	src, srcSrv := newTestServer(t)
	flushOnCleanup(t, src)
	// forrás: titkos adatok, beállítások, sablon, képviselőfotó, leiratkozó, kizárandó fájlok
	pm := src.loadPostmarkSettings()
	pm.From = "Energofish Teszt <teszt@energofish.hu>"
	if err := src.savePostmarkSettings(pm); err != nil {
		t.Fatal(err)
	}
	if err := src.savePostmarkTokens(postmarkTokens{Live: "pm-live-TITKOS-1234", Sandbox: "pm-sandbox-TITKOS"}); err != nil {
		t.Fatal(err)
	}
	photo, _ := os.ReadFile("internal/hirlevel/testdata/foto-minta.webp")
	if code, r := call(t, srcSrv, src.token, "/api/b2b/repphoto", map[string]any{"mono": "GÁL", "data": base64.StdEncoding.EncodeToString(photo)}); code != 200 {
		t.Fatalf("fotó: %v", r)
	}
	fs := src.loadFTPSettings()
	fs.Host, fs.User, fs.RemoteDir = "ftp.pelda.hu", "hirlevel", "/public_html/assets"
	if err := src.saveFTPSettings(fs); err != nil {
		t.Fatal(err)
	}
	if err := src.saveFTPPassword("FTP-TITKOS-jelszo"); err != nil {
		t.Fatal(err)
	}
	if err := src.saveSources(map[string]string{"B2B_HU": "https://pelda.hu/export?token=FORRAS-TITKOS"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src.postmarkFile("leiratkozottak.txt"), []byte("a@pelda.hu\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile("internal/hirlevel/testdata/kerdes-sablon.html")
	if res, err := src.lib.ImportFile("v8-proba.html", raw, nil); err != nil || !res[0].OK {
		t.Fatalf("sablon: %v %+v", err, res)
	}
	for _, n := range []string{"idozites.json", "partnertorzs-b2b_hu.dat"} {
		if err := os.WriteFile(filepath.Join(src.configDir, n), []byte("NEM-KERULHET-AT"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	src.mu.Lock()
	src.state.Content["cover.headline"] = "Átvitt főcím"
	src.state.Output.Dir = filepath.Join(t.TempDir(), "csak-a-forrasgepen")
	src.mu.Unlock()
	_ = os.MkdirAll(src.state.Output.Dir, 0o755)
	outDir := src.state.Output.Dir

	// export: rövid jelszó tilos; a fájlban egyetlen titok sem olvasható
	file := filepath.Join(t.TempDir(), "atvitel")
	if code, r := call(t, srcSrv, src.token, "/api/transfer/export", map[string]any{"password": "rovid", "path": file}); code != 400 || r["error"] == nil {
		t.Fatalf("rövid jelszó: %d %v", code, r)
	}
	code, r := call(t, srcSrv, src.token, "/api/transfer/export", map[string]any{"password": "Kepviselo-2026", "path": file})
	if code != 200 || r["path"] != file+transferExt {
		t.Fatalf("export: %d %v", code, r)
	}
	file += transferExt
	enc, _ := os.ReadFile(file)
	for _, secret := range []string{"pm-live-TITKOS", "FTP-TITKOS", "FORRAS-TITKOS", "Átvitt főcím", "a@pelda.hu"} {
		if bytes.Contains(enc, []byte(secret)) {
			t.Errorf("a fájlban olvasható: %s", secret)
		}
	}
	// a célgépen ezek a mappák nincsenek
	os.RemoveAll(outDir)
	os.RemoveAll(src.loadFTPSettings().LocalDir)

	// cél: van már saját beállítása és leiratkozója
	dst, dstSrv := newTestServer(t)
	flushOnCleanup(t, dst)
	dst.saveNow()
	if err := os.WriteFile(dst.postmarkFile("leiratkozottak.txt"), []byte("b@pelda.hu\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, r := call(t, dstSrv, dst.token, "/api/transfer/inspect", map[string]any{"password": "rossz-jelszo", "path": file}); code != 400 || !strings.Contains(r["error"].(string), "hibás jelszó") {
		t.Fatalf("rossz jelszó: %d %v", code, r)
	}
	code, r = call(t, dstSrv, dst.token, "/api/transfer/inspect", map[string]any{"password": "Kepviselo-2026", "path": file})
	if code != 200 {
		t.Fatalf("inspect: %d %v", code, r)
	}
	sum := r["summary"].(map[string]any)
	if sum["postmarkLive"] != true || sum["ftpPassword"] != true || sum["images"].(float64) != 2 || sum["unsubs"].(float64) != 1 ||
		len(sum["sources"].([]any)) != 1 || len(sum["templates"].([]any)) != 1 || sum["from"] != pm.From {
		t.Errorf("összesítő: %v", sum)
	}
	if strings.Contains(strings.ToLower(toJSON(r)), "titkos") {
		t.Errorf("titok a felületen: %s", toJSON(r))
	}
	id := r["id"].(string)
	code, r = call(t, dstSrv, dst.token, "/api/transfer/apply", map[string]any{"id": id})
	if code != 200 || r["ok"] != true || r["backup"] == "" {
		t.Fatalf("apply: %d %v", code, r)
	}
	if notes, _ := r["notes"].([]any); len(notes) == 0 || !strings.Contains(toJSON(notes), "kimeneti mappa") {
		t.Errorf("megjegyzések: %v", r["notes"])
	}
	// minden átjött
	if tk := dst.loadPostmarkTokens(); tk.Live != "pm-live-TITKOS-1234" || tk.Sandbox != "pm-sandbox-TITKOS" {
		t.Error("Postmark-tokenek")
	}
	if dst.loadPostmarkSettings().From != pm.From {
		t.Error("Postmark-beállítások")
	}
	if dst.loadFTPPassword() != "FTP-TITKOS-jelszo" || dst.loadFTPSettings().Host != "ftp.pelda.hu" {
		t.Error("FTP")
	}
	if dst.loadSources()["B2B_HU"] != "https://pelda.hu/export?token=FORRAS-TITKOS" {
		t.Error("partnerforrás")
	}
	localDir := dst.loadFTPSettings().LocalDir
	if localDir != filepath.Join(dst.configDir, "sajat-kepek") {
		t.Errorf("saját képek mappája: %s", localDir)
	}
	for _, n := range []string{"gal.webp", "gal.png"} {
		if _, err := os.Stat(filepath.Join(localDir, n)); err != nil {
			t.Errorf("hiányzó kép: %s", n)
		}
	}
	dst.mu.Lock()
	head, out := dst.state.Content["cover.headline"], dst.state.Output.Dir
	tpl := dst.lib.All()
	dst.mu.Unlock()
	if head != "Átvitt főcím" {
		t.Errorf("tartalom: %q", head)
	}
	if out != defaultOutputDir() {
		t.Errorf("kimeneti mappa: %s", out)
	}
	found := false
	for _, x := range tpl {
		found = found || x.ID == "v8-proba"
	}
	if !found {
		t.Error("a hozzáadott sablon nem jött át")
	}
	if u := dst.unsubList(); !u["a@pelda.hu"] || !u["b@pelda.hu"] {
		t.Errorf("leiratkozók összefésülése: %v", u)
	}
	for _, n := range []string{"idozites.json", "partnertorzs-b2b_hu.dat"} {
		if _, err := os.Stat(filepath.Join(dst.configDir, n)); err == nil {
			t.Errorf("nem kerülhetett volna át: %s", n)
		}
	}
	if _, err := os.Stat(filepath.Join(r["backup"].(string), "beallitasok.json")); err != nil {
		t.Error("a célgép korábbi beállításairól nincs mentés")
	}
	// a megnyitott csomag egyszer használható
	if code, _ := call(t, dstSrv, dst.token, "/api/transfer/apply", map[string]any{"id": id}); code != 400 {
		t.Error("másodszor is alkalmazható")
	}
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
