package hirlevel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // képformátum
	_ "image/jpeg" // képformátum
	_ "image/png"  // képformátum
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	_ "golang.org/x/image/webp" // képformátum
)

// ImageTarget egy ellenőrizendő kép.
type ImageTarget struct {
	URL   string `json:"url"`
	Where string `json:"where"`
	Kind  string `json:"kind"` // cover, product, portrait, rep, asset
	Key   string `json:"key,omitempty"`
	Index int    `json:"index"`
}

// ImageCheck egy kép ellenőrzésének eredménye.
type ImageCheck struct {
	ImageTarget
	Status  string `json:"status"` // ok, warn, error, skip
	Message string `json:"message"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Bytes   int64  `json:"bytes"`
}

// CollectImageTargets összegyűjti a levélben szereplő képeket.
func CollectImageTargets(c Content, products []Product, partners []Partner, tpl *Template) []ImageTarget {
	var out []ImageTarget
	seen := map[string]bool{}
	add := func(t ImageTarget) {
		if t.URL == "" || seen[t.URL] {
			return
		}
		seen[t.URL] = true
		out = append(out, t)
	}
	d := Build(c, products, &SamplePartner, tpl, "")
	add(ImageTarget{URL: d.Values["cover.image"], Where: "Borítókép", Kind: "cover", Key: "cover.image", Index: -1})
	add(ImageTarget{URL: d.Values["note.signer.portrait"], Where: "Aláíró portréja", Kind: "portrait", Key: "note.signer.portrait", Index: -1})
	sel := 0
	for i, p := range products {
		if !p.On {
			continue
		}
		it := d.Items[sel]
		sel++
		where := fmt.Sprintf("%d. termék", i+1)
		if p.Code != "" {
			where += " (" + p.Code + ")"
		}
		add(ImageTarget{URL: it["image"], Where: where, Kind: "product", Key: "image", Index: i})
	}
	for i := range partners {
		p := &partners[i]
		if strings.TrimSpace(p.RepPhoto) == "" || !ValidPhoto(p.RepPhoto) {
			continue
		}
		pd := Build(c, products, p, tpl, "")
		add(ImageTarget{URL: pd.Values["rep.photo"], Where: "Képviselő fotója: " + p.RepName, Kind: "rep", Key: "repPhoto", Index: i})
	}
	if base := strings.TrimRight(strings.TrimSpace(c["assets.base"]), "/"); base != "" {
		for _, name := range []string{"energofish-mark.png", "energofish-mark-light.png"} {
			add(ImageTarget{URL: base + "/" + name, Where: "Képtár: " + name, Kind: "asset", Key: "assets.base", Index: -1})
		}
		if tpl != nil && tpl.Short == "v2" {
			add(ImageTarget{URL: base + "/wave-band-white.png", Where: "Képtár: wave-band-white.png", Kind: "asset", Key: "assets.base", Index: -1})
			add(ImageTarget{URL: base + "/wave-white-dusk.png", Where: "Képtár: wave-white-dusk.png", Kind: "asset", Key: "assets.base", Index: -1})
		}
		if tpl != nil && tpl.Short == "v4" {
			add(ImageTarget{URL: base + "/contour-band.png", Where: "Képtár: contour-band.png", Kind: "asset", Key: "assets.base", Index: -1})
		}
	}
	return out
}

// CheckImages letölti és ellenőrzi a képeket (párhuzamosan, időkorláttal).
func CheckImages(ctx context.Context, targets []ImageTarget) []ImageCheck {
	out := make([]ImageCheck, len(targets))
	client := &http.Client{Timeout: 20 * time.Second}
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	for i, t := range targets {
		wg.Add(1)
		go func(i int, t ImageTarget) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = checkOne(ctx, client, t)
		}(i, t)
	}
	wg.Wait()
	return out
}

func checkOne(ctx context.Context, client *http.Client, t ImageTarget) ImageCheck {
	r := ImageCheck{ImageTarget: t}
	if !strings.HasPrefix(t.URL, "http://") && !strings.HasPrefix(t.URL, "https://") {
		r.Status, r.Message = "skip", "helyi kép (a képtár feltöltése után ellenőrizhető)"
		return r
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.URL, nil)
	if err != nil {
		r.Status, r.Message = "error", "hibás cím"
		return r
	}
	req.Header.Set("User-Agent", "Energofish-Hirlevel-Generator/1.0")
	resp, err := client.Do(req)
	if err != nil {
		r.Status, r.Message = "error", "nem érhető el: "+shortErr(err)
		return r
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		r.Status, r.Message = "error", fmt.Sprintf("a szerver válasza: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		return r
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		r.Status, r.Message = "error", "letöltési hiba: "+shortErr(err)
		return r
	}
	r.Bytes = int64(len(body))
	cfg, format, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		r.Status, r.Message = "error", "nem kép (vagy nem támogatott formátum): "+resp.Header.Get("Content-Type")
		return r
	}
	r.Width, r.Height = cfg.Width, cfg.Height
	var warns []string
	ratio := float64(cfg.Width) / math.Max(1, float64(cfg.Height))
	square := ratio > 0.92 && ratio < 1.08
	switch t.Kind {
	case "cover":
		if math.Abs(ratio-1200.0/660.0) > 0.04 {
			warns = append(warns, "nem 1200×660 arányú, torzulhat")
		} else if cfg.Width < 1200 {
			warns = append(warns, "kisebb, mint 1200×660, élesebb kép javasolt")
		}
		if r.Bytes > 200*1024 {
			warns = append(warns, fmt.Sprintf("%d KB, javasolt 200 KB alatt", r.Bytes/1024))
		}
	case "product":
		if !square {
			warns = append(warns, "nem négyzetes")
		}
		if cfg.Width < 260 || cfg.Height < 260 {
			warns = append(warns, "kisebb, mint 260×260")
		}
	case "portrait":
		if !square {
			warns = append(warns, "nem négyzetes, torzulhat")
		}
		if cfg.Width < 112 {
			warns = append(warns, "kisebb, mint 112×112")
		}
	case "rep":
		if !square {
			warns = append(warns, "nem négyzetes, torzulhat (64×64-es körben jelenik meg)")
		}
		if cfg.Width < 128 {
			warns = append(warns, "kisebb, mint 128×128, életlen lehet")
		}
		if r.Bytes > 150*1024 {
			warns = append(warns, fmt.Sprintf("%d KB, egy kis körképhez sok", r.Bytes/1024))
		}
	}
	if len(warns) > 0 {
		r.Status, r.Message = "warn", strings.Join(warns, "; ")
	} else {
		r.Status, r.Message = "ok", strings.ToUpper(format)
	}
	return r
}

func shortErr(err error) string {
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	s := err.Error()
	switch {
	case strings.Contains(s, "no such host"):
		return "ismeretlen szerver"
	case strings.Contains(s, "Client.Timeout") || strings.Contains(s, "deadline exceeded"):
		return "időtúllépés"
	case strings.Contains(s, "certificate"):
		return "tanúsítványhiba"
	case strings.Contains(s, "connection refused"):
		return "a kapcsolat elutasítva"
	case strings.Contains(s, "Forbidden"):
		return "a hálózat (proxy/tűzfal) nem engedi"
	}
	if len(s) > 80 {
		s = s[:80] + "…"
	}
	return s
}
