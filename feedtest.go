package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	h "energofish/hirlevel/internal/hirlevel"

	_ "golang.org/x/image/webp"
)

// runFeedTest letölti és feldolgozza az élő cikktörzset, és statisztikát ír a naplóba (CI).
func runFeedTest(url string) error {
	if url == "" {
		url = h.DefaultFeedURL
	}
	dir, err := os.MkdirTemp("", "cikktorzs-teszt")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	s := h.NewFeedStore(dir)
	s.MinInterval = 0
	start := time.Now()
	s.Refresh(url, false)
	last := time.Now()
	for s.Status().Loading {
		time.Sleep(100 * time.Millisecond)
		if time.Since(last) > 3*time.Second {
			st := s.Status()
			log.Printf("  … %s, %.1f MB", st.Phase, float64(st.Bytes)/1e6)
			last = time.Now()
		}
	}
	st := s.Status()
	if !st.Ready {
		return errors.New(st.Error)
	}
	gz, _ := os.Stat(dir + "/termekadatok.xml.gz")
	gzSize := int64(0)
	if gz != nil {
		gzSize = gz.Size()
	}
	log.Printf("cikktörzs: %d cikk, %.1f MB (gyorsítótárban %.1f MB), %s, HTTP %d", st.Count, float64(st.Bytes)/1e6,
		float64(gzSize)/1e6, time.Since(start).Round(time.Millisecond), st.HTTP)

	list := s.Products()
	imgs := map[string]int{}
	var stock [3]int
	sale, noImg, noLink, noDesc, longName := 0, 0, 0, 0, 0
	for i := range list {
		p := &list[i]
		for _, im := range p.Images {
			id := im.ID
			if strings.HasPrefix(id, "gallery") {
				id = "gallery*"
			}
			imgs[id]++
		}
		if p.Stock >= 0 && p.Stock <= 2 {
			stock[p.Stock]++
		}
		if p.RetailAkc > 0 || p.WholesaleAkc > 0 {
			sale++
		}
		if len(p.Images) == 0 {
			noImg++
		}
		if p.Link == "" {
			noLink++
		}
		if h.FeedDesc(p, 30) == "" {
			noDesc++
		}
		if len([]rune(p.Name)) > 26 {
			longName++
		}
	}
	keys := make([]string, 0, len(imgs))
	for k := range imgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := []string{}
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s %d", k, imgs[k]))
	}
	log.Printf("képek: %s; kép nélkül: %d", strings.Join(parts, ", "), noImg)
	log.Printf("készlet: van %d, nincs %d, ismeretlen %d; akciós: %d; link nélkül: %d; leírás nélkül: %d; 26 karakternél hosszabb név: %d",
		stock[1], stock[0], stock[2], sale, noLink, noDesc, longName)

	t0 := time.Now()
	n := 0
	for _, q := range []string{"10000-327", "10000327", "wizard crab", "kamasaki tele", "etetoanyag", "bot"} {
		res := h.SearchFeed(list, q, 60)
		n++
		codes := []string{}
		for i, p := range res {
			if i == 4 {
				break
			}
			codes = append(codes, p.Code+" "+p.Name)
		}
		log.Printf("keresés %-16q → %d találat: %s", q, len(res), strings.Join(codes, " | "))
	}
	log.Printf("keresés átlagosan %s", (time.Since(t0) / time.Duration(n)).Round(time.Microsecond))

	samples := []string{"10000-327", "10729-300", "72151-420", "98015-843", "12049-110"}
	for i := len(list) / 5; i < len(list) && len(samples) < 9; i += max(1, len(list)/5) {
		if !slices.Contains(samples, list[i].Code) {
			samples = append(samples, list[i].Code)
		}
	}
	for _, c := range samples {
		p := h.FindFeed(list, c)
		if p == nil {
			log.Printf("minta %s: nincs a cikktörzsben", c)
			continue
		}
		b, _ := json.Marshal(p.ToProduct(h.FeedOptions{}))
		log.Printf("minta %s → %s", c, b)
	}

	// képméretek mintavétellel
	client := &http.Client{Timeout: 30 * time.Second}
	sizes := map[string][]string{}
	seen := map[string]bool{}
	for _, c := range samples {
		p := h.FindFeed(list, c)
		if p == nil {
			continue
		}
		for _, im := range p.Images {
			if strings.HasPrefix(im.ID, "gallery") && im.ID != "gallery1" {
				continue
			}
			if len(sizes[im.ID]) >= 5 || seen[im.URL] {
				continue
			}
			seen[im.URL] = true
			sizes[im.ID] = append(sizes[im.ID], imageInfo(client, im.URL))
		}
	}
	for _, id := range []string{"thumb", "code", "small", "large", "gallery1"} {
		log.Printf("képméret %-8s %s", id, strings.Join(sizes[id], "; "))
	}

	// feltételes lekérdezés: változatlan feednél 304 a jó válasz
	t1 := time.Now()
	s.Refresh(url, false)
	s.Wait(15 * time.Minute)
	st = s.Status()
	log.Printf("újralekérdezés: HTTP %d, %s, hiba: %q", st.HTTP, time.Since(t1).Round(time.Millisecond), st.Error)
	return nil
}

func imageInfo(c *http.Client, url string) string {
	resp, err := c.Get(url)
	if err != nil {
		return "hiba: " + err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return "hiba: " + err.Error()
	}
	cfg, format, err := image.DecodeConfig(strings.NewReader(string(data)))
	if err != nil {
		return fmt.Sprintf("%d KB, nem kép (%v)", len(data)/1024, err)
	}
	return fmt.Sprintf("%dx%d %s %d KB", cfg.Width, cfg.Height, format, len(data)/1024)
}
