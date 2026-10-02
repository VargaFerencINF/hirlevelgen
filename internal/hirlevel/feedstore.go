package hirlevel

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FeedStore a cikktörzs letöltése, gyorsítótárazása és a keresés adatai.
// A letöltött XML tömörítve a gépre kerül; újraindításkor onnan töltődik, és a
// szerverről csak akkor jön le újra, ha közben változott (ETag / Last-Modified).
type FeedStore struct {
	Dir    string
	Client *http.Client
	// MinInterval: ennyi időn belül nem kérdezi le újra a szervert (kivéve kényszerített frissítés).
	MinInterval time.Duration

	mu        sync.Mutex
	products  []FeedProduct
	loading   bool
	phase     string // cache, download, parse
	bytes     int64
	total     int64
	err       string
	source    string // net, cache
	checkedAt time.Time
	dataTime  time.Time // a cikktörzs letöltésének ideje
	url       string
	http      int // az utolsó letöltés HTTP-állapotkódja (200 / 304)

	pending      bool // frissítés kérése a futó betöltés közben: utána lefut
	pendingURL   string
	pendingForce bool
}

// FeedStatus a felületnek.
type FeedStatus struct {
	Ready     bool   `json:"ready"`
	Loading   bool   `json:"loading"`
	Phase     string `json:"phase,omitempty"`
	Bytes     int64  `json:"bytes"`
	Total     int64  `json:"total"`
	Count     int    `json:"count"`
	Error     string `json:"lastError,omitempty"` // nem „error”: azt a felület API-hibának venné
	Source    string `json:"source,omitempty"`
	DataTime  string `json:"dataTime,omitempty"`
	CheckedAt string `json:"checkedAt,omitempty"`
	URL       string `json:"url"`
	HTTP      int    `json:"http,omitempty"`
}

type feedMeta struct {
	URL          string    `json:"url"`
	ETag         string    `json:"etag"`
	LastModified string    `json:"lastModified"`
	FetchedAt    time.Time `json:"fetchedAt"`
	Count        int       `json:"count"`
}

// NewFeedStore új tár (dir: a gyorsítótár mappája; üres = nincs mentés).
func NewFeedStore(dir string) *FeedStore {
	return &FeedStore{Dir: dir, Client: &http.Client{Timeout: 10 * time.Minute}, MinInterval: 5 * time.Minute}
}

func (s *FeedStore) cachePath() string { return filepath.Join(s.Dir, "termekadatok.xml.gz") }
func (s *FeedStore) metaPath() string  { return filepath.Join(s.Dir, "termekadatok.json") }

// Status az aktuális állapot.
func (s *FeedStore) Status() FeedStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := FeedStatus{Ready: len(s.products) > 0, Loading: s.loading, Phase: s.phase, Bytes: s.bytes, Total: s.total,
		Count: len(s.products), Error: s.err, Source: s.source, URL: s.url, HTTP: s.http}
	if !s.dataTime.IsZero() {
		st.DataTime = s.dataTime.Format(time.RFC3339)
	}
	if !s.checkedAt.IsZero() {
		st.CheckedAt = s.checkedAt.Format(time.RFC3339)
	}
	return st
}

// Products a betöltött cikkek (csak olvasásra).
func (s *FeedStore) Products() []FeedProduct {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.products
}

// Refresh a háttérben frissíti a cikktörzset (ha nem nemrég volt). Ha épp betöltés fut,
// a kérés utána fut le. A visszatérési érték jelzi, hogy lesz-e frissítés.
func (s *FeedStore) Refresh(url string, force bool) bool {
	if url == "" {
		url = DefaultFeedURL
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loading {
		s.pending, s.pendingURL, s.pendingForce = true, url, s.pendingForce || force
		return true
	}
	if !force && s.recentLocked(url) {
		return false
	}
	s.startLocked(url, force, false)
	return true
}

// LoadCached a gépre mentett cikktörzset tölti be a háttérben, hálózat nélkül
// (induláskor, hogy a keresés azonnal működjön).
func (s *FeedStore) LoadCached(url string) {
	if url == "" {
		url = DefaultFeedURL
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loading || len(s.products) > 0 || s.Dir == "" {
		return
	}
	if _, err := os.Stat(s.cachePath()); err != nil {
		return
	}
	s.startLocked(url, false, true)
}

func (s *FeedStore) recentLocked(url string) bool {
	return url == s.url && len(s.products) > 0 && !s.checkedAt.IsZero() && time.Since(s.checkedAt) < s.MinInterval
}

func (s *FeedStore) startLocked(url string, force, cacheOnly bool) {
	s.loading, s.err, s.bytes, s.total = true, "", 0, 0
	go s.run(url, force, cacheOnly)
}

func (s *FeedStore) finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loading, s.phase = false, ""
	if !s.pending {
		return
	}
	url, force := s.pendingURL, s.pendingForce
	s.pending, s.pendingURL, s.pendingForce = false, "", false
	if force || !s.recentLocked(url) {
		s.startLocked(url, force, false)
	}
}

// Wait megvárja a folyamatban lévő frissítést (tesztekhez, parancssorhoz).
func (s *FeedStore) Wait(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		l := s.loading
		s.mu.Unlock()
		if !l {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (s *FeedStore) set(f func()) {
	s.mu.Lock()
	f()
	s.mu.Unlock()
}

func (s *FeedStore) run(url string, force, cacheOnly bool) {
	defer s.finish()
	meta := s.readMeta()
	sameURL := meta.URL == url

	// 1) gyorsítótár betöltése, ha még nincs adat a memóriában
	s.mu.Lock()
	needCache := len(s.products) == 0 || s.url != url
	s.mu.Unlock()
	if needCache && sameURL {
		s.set(func() { s.phase = "cache" })
		if list, err := s.loadCache(); err == nil {
			s.set(func() {
				s.products, s.source, s.url, s.dataTime = list, "cache", url, meta.FetchedAt
			})
		}
	}
	if cacheOnly {
		return
	}

	// 2) feltételes letöltés
	s.set(func() { s.phase = "download" })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		s.set(func() { s.err = "hibás cikktörzs-cím: " + err.Error() })
		return
	}
	req.Header.Set("User-Agent", "Energofish-Hirlevel-Generator")
	s.mu.Lock()
	haveData := len(s.products) > 0 && s.url == url
	s.mu.Unlock()
	if haveData && sameURL && !force {
		if meta.ETag != "" {
			req.Header.Set("If-None-Match", meta.ETag)
		}
		if meta.LastModified != "" {
			req.Header.Set("If-Modified-Since", meta.LastModified)
		}
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		s.set(func() { s.err = "a cikktörzs nem tölthető le: " + shortErr(err); s.checkedAt = time.Now() })
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified && haveData {
		meta.FetchedAt = time.Now()
		s.writeMeta(meta)
		s.set(func() { s.checkedAt, s.dataTime, s.source, s.http = time.Now(), meta.FetchedAt, "net", resp.StatusCode })
		return
	}
	if resp.StatusCode != http.StatusOK {
		s.set(func() {
			s.err = fmt.Sprintf("a cikktörzs nem tölthető le: a szerver válasza %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
			s.checkedAt, s.http = time.Now(), resp.StatusCode
		})
		return
	}
	s.set(func() { s.total, s.http = resp.ContentLength, resp.StatusCode })

	// letöltés közben feldolgozás és tömörített mentés
	var tmpFile *os.File
	var gz *gzip.Writer
	var body io.Reader = &countingReader{r: resp.Body, s: s}
	if s.Dir != "" {
		if err := os.MkdirAll(s.Dir, 0o755); err == nil {
			if f, err := os.CreateTemp(s.Dir, "letoltes-*.tmp"); err == nil {
				tmpFile = f
				gz = gzip.NewWriter(f)
				body = io.TeeReader(body, gz)
			}
		}
	}
	cleanup := func() {
		if tmpFile != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
		}
	}
	list, err := ParseFeed(body)
	if err != nil {
		cleanup()
		s.set(func() { s.err = err.Error(); s.checkedAt = time.Now() })
		return
	}
	_, _ = io.Copy(io.Discard, body) // a fájl végéig, hogy a mentés teljes legyen
	now := time.Now()
	if tmpFile != nil {
		if err := gz.Close(); err == nil {
			tmpFile.Close()
			if err := os.Rename(tmpFile.Name(), s.cachePath()); err == nil {
				s.writeMeta(feedMeta{URL: url, ETag: resp.Header.Get("ETag"), LastModified: resp.Header.Get("Last-Modified"), FetchedAt: now, Count: len(list)})
			} else {
				os.Remove(tmpFile.Name())
			}
		} else {
			cleanup()
		}
	}
	s.set(func() {
		s.products, s.source, s.url, s.dataTime, s.checkedAt = list, "net", url, now, now
	})
}

type countingReader struct {
	r io.Reader
	s *FeedStore
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 {
		c.s.set(func() { c.s.bytes += int64(n) })
	}
	return n, err
}

func (s *FeedStore) readMeta() feedMeta {
	var m feedMeta
	if s.Dir == "" {
		return m
	}
	if b, err := os.ReadFile(s.metaPath()); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	return m
}

func (s *FeedStore) writeMeta(m feedMeta) {
	if s.Dir == "" {
		return
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(s.metaPath(), b, 0o644)
}

func (s *FeedStore) loadCache() ([]FeedProduct, error) {
	if s.Dir == "" {
		return nil, os.ErrNotExist
	}
	f, err := os.Open(s.cachePath())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	return ParseFeed(gz)
}
