package hirlevel

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"path"
	"regexp"
	"strings"

	"golang.org/x/image/webp"
)

// Outlook-barát képek: az asztali Outlook nem jeleníti meg a WEBP képeket. A webes képtárra
// mutató .webp hivatkozások a levélben a képek PNG párjára cserélődnek (x.webp → x.png); a
// párokat az FTP-feltöltés készíti el. A WEBP-ben dolgozhattok tovább, a levélbe PNG kerül.

// KeepWebpKey a közös tartalom kapcsolója: "1" = a WEBP marad a levélben (nincs csere).
// Alapból (üresen) a csere be van kapcsolva.
const KeepWebpKey = "assets.keepWebp"

// OutlookSafeOn igaz, ha a WEBP képek PNG párjukra cserélődnek.
func OutlookSafeOn(c Content) bool { return strings.TrimSpace(c[KeepWebpKey]) != "1" }

// WebpTwin a WEBP fájl PNG párjának neve (x.webp → x.png); ok=false, ha nem WEBP.
func WebpTwin(name string) (string, bool) {
	ext := path.Ext(name)
	if !strings.EqualFold(ext, ".webp") {
		return "", false
	}
	return strings.TrimSuffix(name, ext) + ".png", true
}

// WebpToPNG a WEBP képet PNG-vé alakítja (az átlátszóság megmarad).
func WebpToPNG(data []byte) ([]byte, error) {
	img, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("a WEBP kép nem olvasható: %v", err)
	}
	return encodePNG(img)
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// OutlookSafeURL a webes képtárra (base, http/https) mutató .webp címet a PNG párjára cseréli.
func OutlookSafeURL(u, base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if !(strings.HasPrefix(base, "https://") || strings.HasPrefix(base, "http://")) || !strings.HasPrefix(u, base+"/") {
		return u
	}
	rest, tail := u, ""
	if i := strings.IndexAny(rest, "?#"); i >= 0 {
		rest, tail = rest[:i], rest[i:]
	}
	if twin, ok := WebpTwin(rest); ok {
		return twin + tail
	}
	return u
}

// OutlookSafeHTML a kész levélben a webes képtárra mutató .webp hivatkozásokat (src, url(),
// background) a PNG párjukra cseréli. Más (pl. termékkép-szerver) címhez nem nyúl.
func OutlookSafeHTML(html, base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if !(strings.HasPrefix(base, "https://") || strings.HasPrefix(base, "http://")) {
		return html
	}
	re := regexp.MustCompile(`(` + regexp.QuoteMeta(EscapeHTML(base)) + `/[^"'()\s<>?#]+)\.(?i:webp)(["')?#&\s])`)
	return re.ReplaceAllString(html, "${1}.png${2}")
}

// AddWebpTwins a WEBP fájlok mellé felveszi a PNG párjukat (ha nincs már ilyen nevű fájl a
// képtárban). A hibásan olvasható WEBP-ről figyelmeztetés készül, a feltöltés nem áll le.
func AddWebpTwins(files []AssetFile) ([]AssetFile, []string) {
	have := map[string]bool{}
	for _, f := range files {
		have[strings.ToLower(f.Rel)] = true
	}
	out := append([]AssetFile(nil), files...)
	var warns []string
	for _, f := range files {
		twin, ok := WebpTwin(f.Rel)
		if !ok || have[strings.ToLower(twin)] {
			continue
		}
		src, err := f.read()
		if err == nil {
			src, err = WebpToPNG(src)
		}
		if err != nil {
			warns = append(warns, f.Rel+": "+err.Error())
			continue
		}
		data := src
		out = append(out, AssetFile{Rel: twin, Size: int64(len(data)), Source: f.Source + " (PNG pár)", read: func() ([]byte, error) { return data, nil }})
		have[strings.ToLower(twin)] = true
	}
	return out, warns
}

// RepPhotoSize a programban feltöltött képviselő-fotó mérete (négyzetes, px).
const RepPhotoSize = 192

// CheckRepPhoto a feltöltött képviselő-fotót ellenőrzi (WEBP, legalább 64×64, legfeljebb 2 MB).
func CheckRepPhoto(data []byte) error {
	if len(data) == 0 {
		return errors.New("üres kép")
	}
	if len(data) > 2<<20 {
		return errors.New("a kép túl nagy (legfeljebb 2 MB)")
	}
	cfg, err := webp.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return errors.New("a kép nem WEBP formátumú")
	}
	if cfg.Width < 64 || cfg.Height < 64 {
		return fmt.Errorf("a kép túl kicsi (%d×%d, legalább 64×64 kell)", cfg.Width, cfg.Height)
	}
	return nil
}
