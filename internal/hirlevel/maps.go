package hirlevel

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// A partnerválasztó országtérképei (web/terkepek.json, a tools/terkepek.py állítja elő a
// Natural Earth közkincs adataiból). A partnertörzs Megye mezőjét itt párosítjuk a térkép
// régióival; a felület csak rajzol.

// MapRegion egy megye / régió / ország a térképen.
type MapRegion struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	HU    string   `json:"hu,omitempty"` // magyar név, ha eltér
	D     string   `json:"d,omitempty"`  // SVG-útvonal (sík térkép)
	Keys  []string `json:"keys,omitempty"`
	// G a földgömb országának alakja földrajzi koordinátákkal: sokszögek → gyűrűk → [hossz, szél, …]
	G json.RawMessage `json:"g,omitempty"`
}

// MapGroup több régiót jelölő név (pl. „Vajdaság”, „Cataluña”).
type MapGroup struct {
	Keys []string `json:"keys"`
	IDs  []string `json:"ids"`
}

// CountryMap egy célcsoport térképe.
type CountryMap struct {
	Title   string      `json:"title"`
	Kind    string      `json:"kind"` // „megye”, vagy „globe”: forgó földgömb országokkal (nemzetközi célcsoport)
	ViewBox string      `json:"viewBox"`
	Frame   string      `json:"frame,omitempty"` // kiemelt rész kerete (pl. Kanári-szigetek)
	Regions []MapRegion `json:"regions"`
	Groups  []MapGroup  `json:"groups,omitempty"`

	exact map[string][]string // normalizált név → régiók
}

// MapSet a térképek célcsoportonként.
type MapSet struct {
	Maps map[string]*CountryMap `json:"maps"`
}

// LoadMaps a térképfájl betöltése és a névjegyzékek felépítése.
func LoadMaps(data []byte) (*MapSet, error) {
	var ms MapSet
	if err := json.Unmarshal(data, &ms); err != nil {
		return nil, fmt.Errorf("a térképfájl hibás: %v", err)
	}
	for _, m := range ms.Maps {
		m.index()
	}
	return &ms, nil
}

func (m *CountryMap) index() {
	m.exact = map[string][]string{}
	// a régiók nevei; ha két régió ugyanazt a nevet viseli, a név nem egyértelmű. Előbb a saját
	// (felirat, magyar név, kód), utána a további névalakok: egy régió saját neve erősebb egy
	// másik régió mellékneveinél (pl. „Saint-Martin”).
	owner := map[string]string{}
	primary := map[string]bool{}
	for pass := 0; pass < 2; pass++ {
		for _, r := range m.Regions {
			keys := []string{r.Label, r.HU, r.ID}
			if pass == 1 {
				keys = r.Keys
			}
			for _, k := range keys {
				n := MapKey(k)
				if n == "" {
					continue
				}
				o, ok := owner[n]
				switch {
				case !ok:
					owner[n] = r.ID
					primary[n] = pass == 0
				case o == r.ID:
				case pass == 1 && primary[n]:
					// a másik régió saját neve marad
				default:
					owner[n] = "" // ütközik
				}
			}
		}
	}
	for n, id := range owner {
		if id != "" {
			m.exact[n] = []string{id}
		}
	}
	// régiócsoportok (egy régió saját neve elsőbbséget élvez; a több régióra illő név a csoporté)
	for _, g := range m.Groups {
		ids := append([]string{}, g.IDs...)
		sort.Strings(ids)
		for _, k := range g.Keys {
			if n := MapKey(k); n != "" {
				if o, ok := owner[n]; !ok || o == "" {
					m.exact[n] = ids
				}
			}
		}
	}
}

// mapPartSep az összetett értékek elválasztói („ANDALUCÍA - ALMERÍA”, „Araba/Álava”);
// a szóköz nélküli kötőjel a név része („Borsod-Abaúj-Zemplén”).
var mapPartSep = regexp.MustCompile(`\s+[-–—]+\s+|[/,;()|]`)

// Match a Megye-érték régiói a térképen (üres, ha nem párosítható egyértelműen). Az összetett
// értékeknél („közösség - tartomány”) a legpontosabb, egyetlen régióra illő részt veszi.
func (m *CountryMap) Match(value string) []string {
	if m == nil {
		return nil
	}
	if ids := m.match1(value, false); ids != nil {
		return ids
	}
	parts := mapPartSep.Split(value, -1)
	if len(parts) > 1 {
		if ids := m.matchParts(parts, false); ids != nil {
			return ids
		}
	}
	if ids := m.match1(value, true); ids != nil {
		return ids
	}
	if len(parts) > 1 {
		return m.matchParts(parts, true)
	}
	return nil
}

// matchParts hátulról (a legpontosabb résztől) keres; az egy régióra illő rész nyer, különben
// az első csoport.
func (m *CountryMap) matchParts(parts []string, fuzzy bool) []string {
	var group []string
	for i := len(parts) - 1; i >= 0; i-- {
		ids := m.match1(parts[i], fuzzy)
		if len(ids) == 1 {
			return ids
		}
		if group == nil && len(ids) > 1 {
			group = ids
		}
	}
	return group
}

func (m *CountryMap) match1(value string, fuzzy bool) []string {
	k := MapKey(value)
	if k == "" {
		return nil
	}
	if ids, ok := m.exact[k]; ok {
		return ids
	}
	if !fuzzy {
		return nil
	}
	// névalakok: „Pozsony” ~ „Pozsonyi kerület”, „Bratislava” ~ „Bratislavský”
	cand := map[string][]string{}
	for key, ids := range m.exact {
		short, long := k, key
		if len(short) > len(long) {
			short, long = long, short
		}
		cp := commonPrefix(k, key)
		if (len(short) >= 4 && strings.HasPrefix(long, short)) || (cp >= 6 && cp*10 >= len(short)*7) {
			cand[strings.Join(ids, ",")] = ids
		}
	}
	if len(cand) == 1 {
		for _, ids := range cand {
			return ids
		}
	}
	return nil
}

func commonPrefix(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}

// a nevekből elhagyott általános szavak (megye, kraj, county …)
var mapStopWords = map[string]bool{
	"megye": true, "varmegye": true, "megyei": true, "kerulet": true, "korzet": true, "jaras": true, "tartomany": true,
	"kraj": true, "region": true, "regio": true, "regiunea": true, "county": true, "district": true, "okrug": true,
	"judet": true, "judetul": true, "provincia": true, "province": true, "provincie": true, "land": true, "bundesland": true,
	"state": true, "comunidad": true, "comunitat": true, "autonoma": true, "autonomous": true, "community": true,
	"grad": true, "city": true, "of": true, "the": true, "de": true, "la": true, "las": true, "los": true,
	"municipiul": true, "hlavni": true, "mesto": true, "oblast": true, "pokrajina": true, "autonomna": true,
	"principado": true, "foral": true, "is": true, "islas": true, "illes": true,
}

var mapFold = strings.NewReplacer("ß", "ss", "ł", "l", "Ł", "l", "đ", "d", "Đ", "d", "ø", "o", "Ø", "o", "æ", "ae", "Æ", "ae", "ı", "i", "ô", "o", "Ô", "o")

// MapKey a nevek összevetéshez használt alakja: kisbetűs, ékezet és általános szavak nélkül
// („Pest megye” → „pest”, „Bratislavský kraj” → „bratislavsky”, „Județul Cluj” → „cluj”).
func MapKey(s string) string {
	s = norm.NFD.String(mapFold.Replace(strings.TrimSpace(s)))
	var b strings.Builder
	var words []string
	flush := func() {
		if b.Len() > 0 {
			if w := b.String(); !mapStopWords[w] {
				words = append(words, w)
			}
			b.Reset()
		}
	}
	for _, r := range s {
		switch {
		case unicode.Is(unicode.Mn, r):
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		default:
			flush()
		}
	}
	flush()
	return strings.Join(words, "")
}

// MapMatch a célcsoport Megye-értékeinek párosítása (érték → régiók; a nem párosíthatók
// az unmatched listában).
func (ms *MapSet) MapMatch(group string, values []string) (m *CountryMap, match map[string][]string, unmatched []string) {
	match = map[string][]string{}
	unmatched = []string{}
	if ms == nil {
		return nil, match, unmatched
	}
	m = ms.Maps[group]
	if m == nil {
		return nil, match, unmatched
	}
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			continue
		}
		if ids := m.Match(v); len(ids) > 0 {
			match[v] = ids
		} else {
			unmatched = append(unmatched, v)
		}
	}
	sort.Strings(unmatched)
	return m, match, unmatched
}
