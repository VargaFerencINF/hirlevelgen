package hirlevel

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Content a közös tartalom lapos kulcs → érték formában (pl. "cover.headline").
type Content map[string]string

// Product egy termék az ajánlatban.
type Product struct {
	On    bool   `json:"on"`
	Code  string `json:"code"`
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	Price string `json:"price"`
	Deal  string `json:"deal"`
	Image string `json:"image"`
	URL   string `json:"url"`
	Alt   string `json:"alt,omitempty"`
	CTA   string `json:"cta,omitempty"`
	Row   int    `json:"row,omitempty"` // Excel sor (tájékoztató)
	// Images a cikktörzsben talált képváltozatok (a termékkártyán ezek közül lehet váltani).
	Images []FeedImage `json:"images,omitempty"`
}

// Partner egy címzett az Excel első munkalapjáról.
type Partner struct {
	Row       int               `json:"row"`
	Email     string            `json:"email"`
	Name      string            `json:"name"`
	Company   string            `json:"company"`
	RepName   string            `json:"repName"`
	RepPhoto  string            `json:"repPhoto"`
	RepPhone  string            `json:"repPhone"`
	RepEmail  string            `json:"repEmail"`
	RepRegion string            `json:"repRegion"`
	Extra     map[string]string `json:"extra,omitempty"`     // normalizált oszlopnév → érték (változóként használható)
	Overrides map[string]string `json:"overrides,omitempty"` // sablonkulcs → érték (partnerenkénti felülírás)
}

// SamplePartner az előnézethez, ha még nincs betöltött Excel.
var SamplePartner = Partner{
	Row: 0, Email: "kovacs.peter@example.com", Name: "Kovács Péter", Company: "Minta Horgászbolt",
	RepName: "Nagy Attila", RepPhone: "+36 30 123 4567", RepEmail: "nagy.attila@energofish.hu", RepRegion: "Pest és Nógrád megye",
}

// ---------------------------------------------------------------------------
// Alapértékek

// DefaultContent a tervező mintatartalmából (tartalom-minta.json) készül,
// a partnerenkénti mezők helyén változókkal.
func DefaultContent(sampleJSON []byte) (Content, []Product, error) {
	c, products, err := ImportJSON(sampleJSON)
	if err != nil {
		return nil, nil, err
	}
	set := map[string]string{
		"assets.base":           "",
		"cover.image":           "{assets}/cover-balaton-1200x660.jpg",
		"note.signer.portrait":  "{assets}/portre-helyettesito.png",
		"note.greeting":         "Kedves {nev}!",
		"note.greetingFallback": "Kedves Partnerünk!",
		"meta.preheaderPoll":    "Plusz egy kérdés.",
		"cover.meta.items":      "{termekszam} termék",
		"offer.badge":           "{termekszam} tétel · okt. 31-ig",
		"offer.cta":             "Rendelés",
		"rep.phonePrefix":       "✆",
		"offer.imagePattern":    "https://images.energofish.hu/thumbimage/{cikkszam}.JPG",
		"offer.urlPattern":      "https://b2b.energofish.hu/termek/{cikkszam}",
		"offer.linkParams":      "",
	}
	for k, v := range set {
		c[k] = v
	}
	for i := 1; i <= 3; i++ {
		k := fmt.Sprintf("poll.answers.%d.url", i)
		if c[k] != "" && !strings.Contains(c[k], "{email}") {
			c[k] += "&partner={email}"
		}
	}
	return c, products, nil
}

// Normalize kiegészíti a hiányzó kulcsokat és eldobja az ismeretleneket.
func (c Content) Normalize(defaults Content) Content {
	out := Content{}
	for _, f := range Fields {
		if v, ok := c[f.Key]; ok {
			out[f.Key] = v
		} else {
			out[f.Key] = defaults[f.Key]
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// JSON import / export a tartalom-minta.json szerkezetében

// Flatten a beágyazott JSON-t lapos kulcsokká alakítja. A {text, url} link-objektum
// „kulcs” (text) és „kulcs.url” párrá lesz.
func Flatten(v any, prefix string, out map[string]string) {
	switch t := v.(type) {
	case map[string]any:
		_, hasText := t["text"]
		_, hasURL := t["url"]
		isLink := hasText && hasURL && len(t) == 2
		for k, child := range t {
			key := prefix + "." + k
			if prefix == "" {
				key = k
			}
			if isLink && k == "text" && prefix != "" {
				key = prefix
			}
			Flatten(child, key, out)
		}
	case []any:
		for i, child := range t {
			Flatten(child, prefix+"."+strconv.Itoa(i+1), out)
		}
	case string:
		out[prefix] = t
	case float64:
		out[prefix] = strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		out[prefix] = strconv.FormatBool(t)
	case nil:
		out[prefix] = ""
	}
}

// ImportJSON beolvas egy tartalom-JSON-t (a tervező formátumában).
func ImportJSON(data []byte) (Content, []Product, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("hibás JSON: %w", err)
	}
	flat := map[string]string{}
	Flatten(raw, "", flat)
	c := Content{}
	for _, f := range Fields {
		if v, ok := flat[f.Key]; ok {
			c[f.Key] = v
		}
	}
	// termékek
	idx := map[int]*Product{}
	for k, v := range flat {
		if !strings.HasPrefix(k, "offer.items.") {
			continue
		}
		rest := strings.SplitN(k[len("offer.items."):], ".", 2)
		if len(rest) != 2 {
			continue
		}
		n, err := strconv.Atoi(rest[0])
		if err != nil {
			continue
		}
		p := idx[n]
		if p == nil {
			p = &Product{On: true}
			idx[n] = p
		}
		switch rest[1] {
		case "url":
			p.URL = v
		case "image":
			p.Image = v
		case "imageAlt":
			p.Alt = v
		case "code":
			p.Code = v
		case "name":
			p.Name = v
		case "desc":
			p.Desc = v
		case "price":
			p.Price = v
		case "deal":
			p.Deal = v
		case "cta":
			p.CTA = v
		case "on":
			p.On = v != "false"
		}
	}
	nums := make([]int, 0, len(idx))
	for n := range idx {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	products := make([]Product, 0, len(nums))
	for _, n := range nums {
		products = append(products, *idx[n])
	}
	return c, products, nil
}

// ExportJSON a tartalmat a tervező JSON-formátumában írja ki (termékekkel együtt).
func ExportJSON(c Content, products []Product) ([]byte, error) {
	root := map[string]any{}
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		setPath(root, strings.Split(k, "."), c[k])
	}
	items := map[string]any{}
	for i, p := range products {
		items[strconv.Itoa(i+1)] = map[string]any{
			"on": p.On, "url": p.URL, "image": p.Image, "imageAlt": p.Alt, "code": p.Code, "name": p.Name,
			"desc": p.Desc, "price": p.Price, "deal": p.Deal, "cta": p.CTA,
		}
	}
	offer, _ := root["offer"].(map[string]any)
	if offer == nil {
		offer = map[string]any{}
		root["offer"] = offer
	}
	offer["items"] = items
	return json.MarshalIndent(root, "", "  ")
}

func setPath(m map[string]any, path []string, v string) {
	k := path[0]
	if len(path) == 1 {
		if existing, ok := m[k].(map[string]any); ok {
			existing["text"] = v
		} else {
			m[k] = v
		}
		return
	}
	child, ok := m[k].(map[string]any)
	if !ok {
		child = map[string]any{}
		if s, isStr := m[k].(string); isStr {
			child["text"] = s
		}
		m[k] = child
	}
	setPath(child, path[1:], v)
}

// ---------------------------------------------------------------------------
// Normalizálás, változók

var accentMap = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ö", "o", "ő", "o", "ú", "u", "ü", "u", "ű", "u",
	"Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ö", "o", "Ő", "o", "Ú", "u", "Ü", "u", "Ű", "u",
	"ä", "a", "ô", "o", "õ", "o", "û", "u",
)

// Norm kisbetűs, ékezet nélküli, csak betű-szám alakra hoz (oszlopnevek és változók).
func Norm(s string) string {
	s = strings.ToLower(accentMap.Replace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var tokenRe = regexp.MustCompile(`\{([^{}\n]{1,40})\}`)

// rawTokens értékét URL-ben sem kódoljuk.
var rawTokens = map[string]bool{"assets": true}

// Expand a {változó} tokeneket cseréli. URL-módban az értékeket URL-kódolja
// (a „?” után lekérdezés-, előtte útvonal-kódolással). Visszaadja az ismeretlen
// és az üres értékű változókat is.
func Expand(s string, tokens map[string]string, urlMode bool) (out string, unknown, empty []string) {
	if !strings.Contains(s, "{") {
		return s, nil, nil
	}
	locs := tokenRe.FindAllStringSubmatchIndex(s, -1)
	if len(locs) == 0 {
		return s, nil, nil
	}
	var b strings.Builder
	pos := 0
	for _, m := range locs {
		name := Norm(s[m[2]:m[3]])
		v, ok := tokens[name]
		if !ok {
			unknown = append(unknown, s[m[0]:m[1]])
			continue
		}
		b.WriteString(s[pos:m[0]])
		pos = m[1]
		if strings.TrimSpace(v) == "" {
			empty = append(empty, s[m[0]:m[1]])
		}
		if urlMode && !rawTokens[name] {
			if strings.Contains(s[:m[0]], "?") {
				v = url.QueryEscape(v)
			} else {
				v = url.PathEscape(v)
			}
		}
		b.WriteString(v)
	}
	b.WriteString(s[pos:])
	return b.String(), unknown, empty
}

// PartnerTokens a partner változói.
func PartnerTokens(p *Partner) map[string]string {
	t := map[string]string{}
	for k, v := range p.Extra {
		t[k] = v
	}
	t["nev"] = p.Name
	t["email"] = p.Email
	t["ceg"] = p.Company
	t["kepviselo"] = p.RepName
	t["kepviseloemail"] = p.RepEmail
	t["kepviselotelefon"] = p.RepPhone
	t["terulet"] = p.RepRegion
	return t
}

// ---------------------------------------------------------------------------
// Partner-adatok formázása

// Initials monogram a névből („Nagy Attila” → „NA”, „dr. Kiss Péter” → „KP”).
func Initials(name string) string {
	var words []string
	for _, w := range strings.Fields(name) {
		if strings.HasSuffix(w, ".") || len([]rune(w)) < 2 && !unicode.IsLetter([]rune(w)[0]) {
			continue
		}
		words = append(words, w)
	}
	if len(words) == 0 {
		return ""
	}
	first := []rune(words[0])
	if len(words) == 1 {
		if len(first) >= 2 {
			return strings.ToUpper(string(first[:2]))
		}
		return strings.ToUpper(string(first))
	}
	second := []rune(words[1])
	return strings.ToUpper(string(first[0]) + string(second[0]))
}

var nonDigit = regexp.MustCompile(`[^\d+]`)

// PhoneURL tel: link a telefonszámból (06… → +36…).
func PhoneURL(phone string) string {
	d := nonDigit.ReplaceAllString(phone, "")
	d = strings.TrimLeft(d, " ")
	if d == "" {
		return ""
	}
	plus := strings.HasPrefix(d, "+")
	d = strings.ReplaceAll(d, "+", "")
	switch {
	case plus:
	case strings.HasPrefix(d, "00"):
		d = d[2:]
	case strings.HasPrefix(d, "06"):
		d = "36" + d[2:]
	case strings.HasPrefix(d, "36") && len(d) >= 10:
	default:
		d = "36" + strings.TrimLeft(d, "0")
	}
	return "tel:+" + d
}

// PhoneDisplay a számot olvasható alakra hozza, ha csak számjegyekből áll
// (36301234567 → +36 30 123 4567); a kézzel tagolt számot meghagyja.
func PhoneDisplay(phone string) string {
	s := strings.TrimSpace(phone)
	if s == "" || strings.ContainsAny(s, " -/()") {
		return s
	}
	u := PhoneURL(s)
	d := strings.TrimPrefix(u, "tel:+")
	if !strings.HasPrefix(d, "36") {
		return s
	}
	n := d[2:]
	switch {
	case len(n) == 9 && n[0] != '1': // +36 30 123 4567
		return "+36 " + n[:2] + " " + n[2:5] + " " + n[5:]
	case len(n) == 8 && n[0] == '1': // +36 1 234 5678
		return "+36 1 " + n[1:4] + " " + n[4:]
	case len(n) == 8: // +36 94 123 456
		return "+36 " + n[:2] + " " + n[2:5] + " " + n[5:]
	}
	return s
}

var plainNumber = regexp.MustCompile(`^\s*(\d{1,12})([.,]\d+)?\s*(Ft|HUF|ft)?\s*$`)

// FormatPrice a csupasz számot „3 090 Ft” alakra hozza (nem törő szóközzel).
func FormatPrice(s string) string {
	m := plainNumber.FindStringSubmatch(s)
	if m == nil {
		return strings.TrimSpace(s)
	}
	f, err := strconv.ParseFloat(m[1]+strings.ReplaceAll(m[2], ",", "."), 64)
	if err != nil {
		return strings.TrimSpace(s)
	}
	n := int64(f + 0.5)
	digits := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteRune('\u00a0')
		}
		b.WriteRune(r)
	}
	return b.String() + "\u00a0Ft"
}

// AppendParams URL-hez fűz lekérdezés-paramétereket (a # rész elé).
func AppendParams(u, params string) string {
	params = strings.TrimLeft(strings.TrimSpace(params), "?&")
	if params == "" || u == "" || !(strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")) {
		return u
	}
	frag := ""
	if i := strings.IndexByte(u, '#'); i >= 0 {
		u, frag = u[:i], u[i:]
	}
	sep := "?"
	if strings.Contains(u, "?") {
		sep = "&"
		if strings.HasSuffix(u, "?") || strings.HasSuffix(u, "&") {
			sep = ""
		}
	}
	return u + sep + params + frag
}
