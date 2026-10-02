package hirlevel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// B2BStore a célcsoportok helyi adatbázisai (titkosítva, ha a platform tudja) és a letöltés.
type B2BStore struct {
	Dir string
	// Protect / Unprotect: a fájlok titkosítása (Windowson DPAPI, a felhasználói fiókhoz kötve).
	Protect   func([]byte) ([]byte, error)
	Unprotect func([]byte) ([]byte, error)
	Client    *http.Client
	// MinRatio: ha az export ennél kevesebb partnert ad a jelenlegi aktívakhoz képest, megáll (S01b).
	MinRatio   float64
	Retries    int
	RetryDelay time.Duration
	Now        func() time.Time

	mu  sync.Mutex
	dbs map[string]*B2BDB
}

// NewB2BStore új tár.
func NewB2BStore(dir string) *B2BStore {
	return &B2BStore{Dir: dir, Client: &http.Client{Timeout: 60 * time.Second}, MinRatio: 0.8, Retries: 3,
		RetryDelay: 2 * time.Second, Now: time.Now, dbs: map[string]*B2BDB{}}
}

const protectedMagic = "EFPROT1\n"

func (s *B2BStore) path(group string) string {
	return filepath.Join(s.Dir, "partnertorzs-"+strings.ToLower(group)+".dat")
}

// DB a célcsoport adatbázisa (üres, ha még nem volt szinkron).
func (s *B2BStore) DB(group string) (*B2BDB, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dbLocked(group)
}

func (s *B2BStore) dbLocked(group string) (*B2BDB, error) {
	if db, ok := s.dbs[group]; ok {
		return db, nil
	}
	db := &B2BDB{Version: 1, Group: group}
	if s.Dir != "" {
		data, err := os.ReadFile(s.path(group))
		switch {
		case err == nil:
			if bytes.HasPrefix(data, []byte(protectedMagic)) {
				if s.Unprotect == nil {
					return nil, errors.New("a partnertörzs titkosított, ezen a gépen nem olvasható")
				}
				if data, err = s.Unprotect(data[len(protectedMagic):]); err != nil {
					return nil, fmt.Errorf("a partnertörzs nem fejthető vissza (másik felhasználó vagy gép?): %v", err)
				}
			}
			if err := json.Unmarshal(data, db); err != nil {
				return nil, fmt.Errorf("a helyi partnertörzs sérült: %v", err)
			}
			db.Group = group
		case !errors.Is(err, os.ErrNotExist):
			return nil, err
		}
	}
	s.dbs[group] = db
	return db, nil
}

func (s *B2BStore) saveLocked(db *B2BDB) error {
	if s.Dir == "" {
		return nil
	}
	data, err := json.Marshal(db)
	if err != nil {
		return err
	}
	if s.Protect != nil {
		enc, err := s.Protect(data)
		if err != nil {
			return fmt.Errorf("a partnertörzs titkosítása nem sikerült: %v", err)
		}
		data = append([]byte(protectedMagic), enc...)
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	tmp := s.path(db.Group) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path(db.Group))
}

// SyncResult egy szinkron eredménye.
type SyncResult struct {
	Log    ImportLog    `json:"log"`
	Report ExportReport `json:"report"`
}

// Sync letölti az exportot és frissíti az adatbázist (S01–S09). Hiba esetén semmi nem
// változik, csak a napló; az ErrSuspiciousExport hibánál force-szal mégis lefuttatható.
func (s *B2BStore) Sync(ctx context.Context, group, src string, force bool) (SyncResult, error) {
	if strings.TrimSpace(src) == "" {
		return SyncResult{}, errors.New("ehhez a célcsoporthoz nincs megadva forrás (token)")
	}
	start := s.Now()
	data, status, ferr := s.fetch(ctx, src)
	return s.apply(group, data, status, ferr, start, force, "")
}

// ImportData egy kézzel letöltött (pl. böngészőben megnyitott) export betöltése ugyanazokkal a
// szabályokkal, mint a szinkron. A source a napló számára rövid leírás (pl. a fájl neve).
func (s *B2BStore) ImportData(group string, data []byte, source string, force bool) (SyncResult, error) {
	return s.apply(group, data, 0, nil, s.Now(), force, source)
}

func (s *B2BStore) apply(group string, data []byte, status int, ferr error, start time.Time, force bool, source string) (SyncResult, error) {
	var res SyncResult
	s.mu.Lock()
	defer s.mu.Unlock()
	db, err := s.dbLocked(group)
	if err != nil {
		return res, err
	}
	fail := func(cause error, rep *ExportReport) (SyncResult, error) {
		next := db.Clone()
		res.Log = next.LogFailure(start, status, cause.Error(), rep)
		res.Log.Source = source
		next.Log[len(next.Log)-1].Source = source
		if s.saveLocked(next) == nil {
			s.dbs[group] = next
		}
		return res, cause
	}
	if ferr != nil {
		return fail(ferr, nil)
	}
	list, rep, err := ParseB2BExport(data)
	res.Report = rep
	if err != nil {
		return fail(err, &rep)
	}
	next := db.Clone()
	l, err := next.Apply(list, rep, start, s.MinRatio, force)
	if err != nil {
		return fail(err, &rep)
	}
	l.HTTP, l.Source = status, source
	next.Log[len(next.Log)-1] = l
	if err := s.saveLocked(next); err != nil { // S08: csak sikeres mentés után cseréljük
		return res, fmt.Errorf("a partnertörzs mentése nem sikerült, semmi nem változott: %v", err)
	}
	s.dbs[group] = next
	res.Log = l
	return res, nil
}

// fetch letöltés újrapróbálkozással; a hibaüzenetben soha nincs benne a cím (token).
func (s *B2BStore) fetch(ctx context.Context, src string) ([]byte, int, error) {
	var lastErr error
	status := 0
	for attempt := 0; attempt <= s.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, status, errors.New("megszakítva")
			case <-time.After(s.RetryDelay * time.Duration(1<<(attempt-1))):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
		if err != nil {
			return nil, 0, errors.New("hibás forráscím")
		}
		req.Header.Set("User-Agent", "Energofish-Hirlevel-Generator")
		req.Header.Set("Accept", "application/json")
		resp, err := s.Client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("a partnertörzs nem tölthető le: %s", SafeNetErr(err))
			continue
		}
		status = resp.StatusCode
		body, rerr := io.ReadAll(io.LimitReader(resp.Body, 200<<20))
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("a szerver hibát adott (HTTP %d)", resp.StatusCode)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, status, fmt.Errorf("a szerver válasza HTTP %d (érvénytelen vagy lejárt token?)", resp.StatusCode)
		}
		if rerr != nil {
			lastErr = fmt.Errorf("a letöltés megszakadt: %s", SafeNetErr(rerr))
			continue
		}
		return body, status, nil
	}
	return nil, status, lastErr
}

// SafeNetErr hálózati hiba a cím nélkül (a cím a tokent tartalmazza).
func SafeNetErr(err error) string {
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	msg := shortErr(err)
	return MaskSecrets(msg)
}

var secretRx = regexp.MustCompile(`(?i)(token=|[?&]v=)[0-9a-z]{8,}|\b[0-9a-f]{32}\b`)

// MaskSecrets minden tokennek látszó részt kitakar (naplóba, hibaüzenetbe kerülő szövegekhez).
func MaskSecrets(s string) string {
	return secretRx.ReplaceAllStringFunc(s, func(m string) string {
		if i := strings.Index(m, "="); i >= 0 {
			return m[:i+1] + "***"
		}
		return "***"
	})
}

// MaskToken a token rövid, felismerhető alakja a felületre („8d60…3f96”).
func MaskToken(tok string) string {
	if len(tok) < 10 {
		return "***"
	}
	return tok[:4] + "…" + tok[len(tok)-4:]
}

// ParseB2BSources a beillesztett szövegből kiolvassa a célcsoportok forráscímeit
// (pl. „B2B HU: https://…&token=…” soronként). Csak tokent tartalmazó címet fogad el.
func ParseB2BSources(text string) map[string]string {
	out := map[string]string{}
	lineRx := regexp.MustCompile(`(?i)B2B[\s_-]*(HU|SK|CZ|COM|AT|DE|RO|ES|RS)\b[^\n]*?(https?://\S+)`)
	for _, m := range lineRx.FindAllStringSubmatch(text, -1) {
		if u := NormalizeB2BSource(m[2]); u != "" {
			out["B2B_"+strings.ToUpper(m[1])] = u
		}
	}
	return out
}

// NormalizeB2BSource a forrás teljes címe: egy puszta tokenből a szabványos cím,
// egy címből változatlanul (ha van benne token). Érvénytelen bemenetre üres.
func NormalizeB2BSource(s string) string {
	s = strings.Trim(strings.TrimSpace(s), "<>\"'")
	if tokenRx.MatchString(strings.ToLower(s)) {
		return B2BExportURL + strings.ToLower(s)
	}
	if !strings.HasPrefix(s, "https://") && !strings.HasPrefix(s, "http://") {
		return ""
	}
	if TokenFromURL(s) == "" {
		return ""
	}
	return s
}

// MaskSource a forráscím a felületre: a token kitakarva („energofish.hu · token 8d60…3f96”).
func MaskSource(src string) string {
	u, err := url.Parse(src)
	if err != nil {
		return "***"
	}
	return u.Host + " · token " + MaskToken(TokenFromURL(src))
}

// TokenFromURL a forráscím token paramétere (vagy maga a token, ha csak az van megadva).
func TokenFromURL(s string) string {
	s = strings.TrimSpace(s)
	if tokenRx.MatchString(strings.ToLower(s)) {
		return strings.ToLower(s)
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	tok := strings.TrimSpace(u.Query().Get("token"))
	if regexp.MustCompile(`^[0-9A-Za-z]{16,}$`).MatchString(tok) {
		return tok
	}
	return ""
}
