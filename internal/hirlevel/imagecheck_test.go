package hirlevel

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"image/png"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckImagesProduct(t *testing.T) {
	enc := func(w, h int, noise bool) []byte {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		if noise { // tömöríthetetlen tartalom a nagy fájlmérethez
			_, _ = rand.New(rand.NewSource(1)).Read(img.Pix)
		}
		var b bytes.Buffer
		if noise {
			_ = png.Encode(&b, img)
		} else {
			_ = jpeg.Encode(&b, img, nil)
		}
		return b.Bytes()
	}
	files := map[string][]byte{
		"/fekvo.jpg":  enc(600, 400, false),
		"/kicsi.jpg":  enc(150, 150, false),
		"/csik.jpg":   enc(900, 300, false),
		"/nehez.png":  enc(500, 500, true),
		"/nemkep.jpg": []byte("<html>"),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(b)
	}))
	defer srv.Close()
	var targets []ImageTarget
	for _, p := range []string{"/fekvo.jpg", "/kicsi.jpg", "/csik.jpg", "/nehez.png", "/nemkep.jpg", "/nincs.jpg"} {
		targets = append(targets, ImageTarget{URL: srv.URL + p, Kind: "product"})
	}
	res := CheckImages(context.Background(), targets)
	want := []struct{ status, msg string }{
		{"ok", "JPEG"}, // a fekvő termékfotó is megfelel (a sablon középre igazítja)
		{"warn", "260 px"},
		{"warn", "elnyújtott"},
		{"warn", "KB"},
		{"error", "nem kép"},
		{"error", "404"},
	}
	for i, w := range want {
		if res[i].Status != w.status || !strings.Contains(res[i].Message, w.msg) {
			t.Errorf("%s: %s %q", targets[i].URL, res[i].Status, res[i].Message)
		}
	}
}

func TestCheckURLWebp(t *testing.T) {
	if lvl, msg := checkURL("https://images.energofish.hu/smallimage/S12049-110_AI.WEBP", "image"); lvl != LevelWarn || !strings.Contains(msg, "Outlook") {
		t.Errorf("webp: %s %s", lvl, msg)
	}
	if lvl, _ := checkURL("https://x.hu/a.webp?v=1", "url"); lvl != "" {
		t.Error("a gomb linkre nem vonatkozik")
	}
	if lvl, _ := checkURL("https://x.hu/a.jpg", "image"); lvl != "" {
		t.Error("jpg")
	}
	// többszörös szóköz a cikknévben
	p := convertTermek(&xmlTermek{CikkszamK: "M8185-6RD", Termeknev: " MUSTAD VERTABRATA 70S 10G SARDINE \u00a0SARDINE "})
	if p.Name != "MUSTAD VERTABRATA 70S 10G SARDINE SARDINE" {
		t.Errorf("%q", p.Name)
	}
}
