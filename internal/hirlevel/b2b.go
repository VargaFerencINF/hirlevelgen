package hirlevel

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// A B2B partner-címtörzs (Webgalamb export, JSON) feldolgozása és szinkronja.
// Specifikáció: energofish_b2b_hu_partnertorzs_spec2.xlsx (S01–S12 szinkronszabályok,
// V01–V20 ellenőrzések). Biztonság: a forrás-token és a partnerek Token mezője titkos,
// a Leiratkozas_link pedig SOHA nem hívható meg (azonnal leiratkoztatna).

// B2BGroup egy célcsoport (külön export, külön forrás-token).
type B2BGroup struct {
	ID      string `json:"id"`      // B2B_HU
	Label   string `json:"label"`   // B2B HU
	Country string `json:"country"` // Magyarország
	Env     string `json:"env"`     // környezeti változó a tokenhez
}

// B2BGroups a célcsoportok (most a HU a fókusz, a többi azonos szerkezetű).
var B2BGroups = []B2BGroup{
	{"B2B_HU", "B2B HU", "Magyarország", ""},
	{"B2B_SK", "B2B SK", "Szlovákia", ""},
	{"B2B_CZ", "B2B CZ", "Csehország", ""},
	{"B2B_COM", "B2B COM", "Nemzetközi (.com)", ""},
	{"B2B_AT", "B2B AT", "Ausztria", ""},
	{"B2B_DE", "B2B DE", "Németország", ""},
	{"B2B_RO", "B2B RO", "Románia", ""},
	{"B2B_ES", "B2B ES", "Spanyolország", ""},
	{"B2B_RS", "B2B RS", "Szerbia", ""},
}

func init() {
	for i := range B2BGroups {
		B2BGroups[i].Env = "WEBGALAMB_TOKEN_" + B2BGroups[i].ID
	}
}

// FindB2BGroup azonosító vagy felirat alapján („B2B_HU”, „B2B HU”, „hu”).
func FindB2BGroup(s string) *B2BGroup {
	k := Norm(s)
	for i := range B2BGroups {
		g := &B2BGroups[i]
		if k == Norm(g.ID) || k == Norm(strings.TrimPrefix(g.ID, "B2B_")) {
			return g
		}
	}
	return nil
}

// B2BExportURL a forrás címe a tokenből.
const B2BExportURL = "https://energofish.hu/admintool/webgalamb_mod.php?action=export&token="

// B2BPartner egy partner a saját adatbázisban (a forrásmezők + a szinkron állapota).
type B2BPartner struct {
	Email       string            `json:"email"` // kulcs (kisbetűs)
	Nazon       string            `json:"nazon"`
	Name        string            `json:"name"`
	Phone       string            `json:"phone,omitempty"`
	Subscribed  string            `json:"subscribed,omitempty"` // „2006-01-02 15:04:05”; érvénytelennél üres
	Unsubscribe string            `json:"unsubscribe"`          // SOHA nem hívható meg, csak a levélbe kerül
	RepName     string            `json:"repName,omitempty"`
	RepPhone    string            `json:"repPhone,omitempty"`
	RepEmail    string            `json:"repEmail,omitempty"`
	RepMono     string            `json:"repMono,omitempty"`
	Shop        string            `json:"shop,omitempty"` // Horgászbolt / Partnerbolt / üres (nincs adat)
	Fix         bool              `json:"fix"`            // belső (Energofish) másolati cím
	Commission  bool              `json:"commission"`     // bizományos profil
	Level       string            `json:"level"`          // Besor
	County      string            `json:"county"`
	WGStatus    string            `json:"wgStatus"`
	Prop6       string            `json:"prop6,omitempty"`
	Token       string            `json:"token,omitempty"`  // TITKOS: automatikus bejelentkezés
	Extra       map[string]string `json:"extra,omitempty"`  // ismeretlen (új) mezők
	NoMail      string            `json:"noMail,omitempty"` // ha nem kaphat levelet: az oka

	Active      bool       `json:"active"`
	FirstImport time.Time  `json:"firstImport"`
	LastImport  time.Time  `json:"lastImport"`
	Inactivated *time.Time `json:"inactivated,omitempty"`
	InactiveWhy string     `json:"inactiveWhy,omitempty"`
}

// sameSource igaz, ha a forrásból jövő mezők azonosak.
func (p *B2BPartner) sameSource(o *B2BPartner) bool {
	a, b := *p, *o
	for _, x := range []*B2BPartner{&a, &b} {
		x.Active, x.FirstImport, x.LastImport, x.Inactivated, x.InactiveWhy = false, time.Time{}, time.Time{}, nil, ""
	}
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	return bytes.Equal(ja, jb)
}

func (p *B2BPartner) copySource(o *B2BPartner) {
	keep := *p
	*p = *o
	p.Active, p.FirstImport, p.LastImport, p.Inactivated, p.InactiveWhy = keep.Active, keep.FirstImport, keep.LastImport, keep.Inactivated, keep.InactiveWhy
}

// Mailable igaz, ha a partner kaphat levelet (aktív és nincs akadálya).
func (p *B2BPartner) Mailable() bool { return p.Active && p.NoMail == "" }

// ExportReport az export ellenőrzésének eredménye (személyes adat és token nélkül).
type ExportReport struct {
	Records     int      `json:"records"` // a letöltött rekordok
	Unique      int      `json:"unique"`  // összevonás után
	Skipped     int      `json:"skipped"` // hibás, kihagyott rekordok
	Merged      int      `json:"merged"`  // az exporton belüli duplikátumok
	NoToken     int      `json:"noToken"` // „Torolt” vagy hibás token
	NoRep       int      `json:"noRep"`   // nincs területi képviselő
	NoMail      int      `json:"noMail"`  // nem kaphat levelet (pl. hibás leiratkozó link)
	UnknownKeys []string `json:"unknownKeys,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

func (r *ExportReport) warn(format string, a ...any) {
	if len(r.Warnings) < 40 {
		r.Warnings = append(r.Warnings, fmt.Sprintf(format, a...))
	}
}

var b2bRequired = []string{
	"Email_cim", "Nazon", "Nev", "Telefonszam", "Feliratkozas_datum", "Leiratkozas_link",
	"Teruleti_kepviselo_nev", "Teruleti_kepviselo_telefonszam", "Teruleti_kepviselo_email_cim",
	"Teruleti_kepviselo_monogram", "Fix", "Bizomanyos", "Besor", "Megye", "Statusz", "Tulajdonsag_6", "Token",
}

var b2bKnown = func() map[string]bool {
	m := map[string]bool{"Partnerbolt_statusz": true}
	for _, k := range b2bRequired {
		m[k] = true
	}
	return m
}()

var (
	emailRx  = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[A-Za-z]{2,}$`)
	tokenRx  = regexp.MustCompile(`^[0-9a-f]{32}$`)
	unsubRx  = regexp.MustCompile(`^https://energofish\.hu/leiratkozas\.html\?&c=[0-9a-f]{20}$`)
	b2bClean = func(s string) string { return strings.TrimFunc(s, unicode.IsSpace) }
)

// parseYesNo: LANG_ADMIN_YES / igen → true, LANG_ADMIN_NO / nem → false (V12).
func parseYesNo(s string) (bool, bool) {
	switch strings.ToLower(b2bClean(s)) {
	case "lang_admin_yes", "igen", "yes", "true", "1":
		return true, true
	case "lang_admin_no", "nem", "no", "false", "0":
		return false, true
	}
	return false, false
}

func jsonString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return x.String()
	case bool:
		return strconv.FormatBool(x)
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}

// ParseB2BExport a Webgalamb exportot ellenőrzi, normalizálja és összevonja (S02–S04).
// Hibát ad (és semmi nem változhat), ha az export nem JSON tömb, üres, vagy túl sok a hibás rekord.
func ParseB2BExport(data []byte) ([]B2BPartner, ExportReport, error) {
	var rep ExportReport
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var raw []map[string]any
	if err := dec.Decode(&raw); err != nil {
		trim := bytes.TrimSpace(data)
		if len(trim) > 0 && trim[0] != '[' {
			return nil, rep, errors.New("a válasz nem JSON tömb (lehet, hogy a token érvénytelen vagy lejárt)")
		}
		return nil, rep, fmt.Errorf("a válasz nem értelmezhető JSON: %v", err)
	}
	rep.Records = len(raw)
	if len(raw) == 0 {
		return nil, rep, errors.New("az export üres (0 partner) – biztonsági okból nem dolgozom fel")
	}
	unknown := map[string]bool{}
	byEmail := map[string]int{}
	var out []B2BPartner
	missingKeys := 0
	for i, obj := range raw {
		get := func(k string) string { return b2bClean(jsonString(obj[k])) }
		var missing []string
		for _, k := range b2bRequired {
			if _, ok := obj[k]; !ok {
				missing = append(missing, k)
			}
		}
		nazon := get("Nazon")
		if len(missing) > 0 {
			rep.Skipped++
			missingKeys++
			rep.warn("%d. rekord (%s): hiányzó mező: %s – kihagyva", i+1, nazon, strings.Join(missing, ", "))
			continue
		}
		p := B2BPartner{
			Email: strings.ToLower(get("Email_cim")), Nazon: nazon, Name: get("Nev"), Phone: get("Telefonszam"),
			Unsubscribe: get("Leiratkozas_link"),
			RepName:     get("Teruleti_kepviselo_nev"), RepPhone: get("Teruleti_kepviselo_telefonszam"),
			RepEmail: strings.ToLower(get("Teruleti_kepviselo_email_cim")), RepMono: strings.ToUpper(get("Teruleti_kepviselo_monogram")),
			Shop: get("Partnerbolt_statusz"), Level: strings.ToUpper(get("Besor")), County: get("Megye"),
			WGStatus: get("Statusz"), Prop6: get("Tulajdonsag_6"),
		}
		for k, v := range obj {
			if !b2bKnown[k] {
				unknown[k] = true
				if p.Extra == nil {
					p.Extra = map[string]string{}
				}
				p.Extra[k] = b2bClean(jsonString(v))
			}
		}
		if !emailRx.MatchString(p.Email) { // V01
			rep.Skipped++
			rep.warn("%d. rekord (%s): érvénytelen e-mail cím – kihagyva", i+1, nazon)
			continue
		}
		var ok1, ok2 bool
		p.Fix, ok1 = parseYesNo(get("Fix"))
		p.Commission, ok2 = parseYesNo(get("Bizomanyos"))
		if !ok1 || !ok2 { // V12
			rep.Skipped++
			rep.warn("%d. rekord (%s): ismeretlen Fix/Bizomanyos érték – kihagyva", i+1, nazon)
			continue
		}
		if d := get("Feliratkozas_datum"); d != "" { // V15
			if _, err := time.Parse("2006-01-02 15:04:05", d); err == nil {
				p.Subscribed = d
			} else {
				rep.warn("%s: érvénytelen feliratkozási dátum", nazon)
			}
		}
		if tok := strings.ToLower(get("Token")); tokenRx.MatchString(tok) { // V05
			p.Token = tok
		} else {
			rep.NoToken++
		}
		switch u := p.Unsubscribe; { // V07
		case u == "" || !strings.HasPrefix(u, "https://") || strings.ContainsAny(u, " \t\"<>"):
			p.NoMail = "nincs érvényes leiratkozó link"
		case !unsubRx.MatchString(u) && strings.Contains(u, "energofish.hu/leiratkozas"):
			rep.warn("%s: a leiratkozó link szokatlan formátumú", nazon)
		}
		if p.WGStatus != "Aktív" { // V13
			rep.warn("%s: a forrásrendszer státusza „%s”", nazon, p.WGStatus)
			if p.NoMail == "" {
				p.NoMail = "a forrásrendszer státusza: " + p.WGStatus
			}
		}
		if p.RepMono == "" && p.RepName == "" { // V09
			rep.NoRep++
		}
		if j, dup := byEmail[p.Email]; dup { // S04: összevonás
			rep.Merged++
			prev := &out[j]
			fix := prev.Fix || p.Fix
			first := prev.Subscribed
			if p.Subscribed != "" && (first == "" || p.Subscribed < first) {
				first = p.Subscribed
			}
			if p.Subscribed >= prev.Subscribed { // a többi mező a legutóbbi feliratkozásból
				*prev = p
			}
			prev.Fix, prev.Subscribed = fix, first
			continue
		}
		byEmail[p.Email] = len(out)
		out = append(out, p)
	}
	if missingKeys >= 3 && missingKeys*20 > len(raw) { // S02: >5% hibás szerkezetű rekord (szerkezetváltozás)
		return nil, rep, fmt.Errorf("az export %d rekordjából %d-ből hiányoznak kötelező mezők – a szerkezet megváltozott, a betöltés megszakítva", len(raw), missingKeys)
	}
	if len(out) == 0 {
		return nil, rep, errors.New("az exportban nincs egyetlen érvényes partner sem")
	}
	// V06: ugyanaz a token több e-mailnél → biztonsági kockázat, a tokent nem használjuk
	tokens := map[string][]int{}
	for i := range out {
		if out[i].Token != "" {
			tokens[out[i].Token] = append(tokens[out[i].Token], i)
		}
	}
	for _, idx := range tokens {
		if len(idx) > 1 {
			for _, i := range idx {
				out[i].Token = ""
			}
			rep.warn("ugyanaz a partner-token %d e-mail címnél szerepel – ezeknél a token nem használható", len(idx))
		}
	}
	// V10: monogram → név/e-mail/telefon nem 1:1
	reps := map[string]string{}
	for i := range out {
		p := &out[i]
		if p.RepMono == "" {
			continue
		}
		sig := p.RepName + "|" + p.RepEmail + "|" + p.RepPhone
		if prev, ok := reps[p.RepMono]; ok && prev != sig {
			rep.warn("a(z) %s monogramhoz eltérő képviselő-adatok tartoznak", p.RepMono)
			reps[p.RepMono] = sig
		} else if !ok {
			reps[p.RepMono] = sig
		}
	}
	for i := range out {
		if out[i].NoMail != "" {
			rep.NoMail++
		}
	}
	for k := range unknown {
		rep.UnknownKeys = append(rep.UnknownKeys, k)
	}
	sort.Strings(rep.UnknownKeys)
	rep.Unique = len(out)
	sort.Slice(out, func(i, j int) bool { return out[i].Email < out[j].Email })
	return out, rep, nil
}

// ---------------------------------------------------------------------------
// Saját adatbázis és szinkron (S05–S08)

// B2BDB egy célcsoport partnerei és importnaplója.
type B2BDB struct {
	Version  int           `json:"version"`
	Group    string        `json:"group"`
	Partners []*B2BPartner `json:"partners"`
	Log      []ImportLog   `json:"log"`
	SyncedAt time.Time     `json:"syncedAt"` // az utolsó sikeres szinkron
}

// ImportLog egy szinkronfutás (S09: csak darabszámok, személyes adat és token nélkül).
type ImportLog struct {
	ID          int       `json:"id"`
	Group       string    `json:"group"`
	Start       time.Time `json:"start"`
	HTTP        int       `json:"http,omitempty"`
	Source      string    `json:"source,omitempty"` // kézi betöltésnél a fájl neve (egyébként a letöltés)
	Records     int       `json:"records"`
	Unique      int       `json:"unique"`
	New         int       `json:"new"`
	Updated     int       `json:"updated"`
	Changed     int       `json:"changed"` // ebből ténylegesen változott adat
	Reactivated int       `json:"reactivated"`
	Inactivated int       `json:"inactivated"`
	Active      int       `json:"active"` // aktív partnerek a futás után
	Unknown     []string  `json:"unknownKeys,omitempty"`
	OK          bool      `json:"ok"`
	Result      string    `json:"result"`
	Warnings    []string  `json:"warnings,omitempty"`
}

// ErrSuspiciousExport: az export gyanúsan kevés partnert tartalmaz (S01b).
var ErrSuspiciousExport = errors.New("gyanúsan kevés partner")

// ActiveCount az aktív partnerek száma.
func (db *B2BDB) ActiveCount() int {
	n := 0
	for _, p := range db.Partners {
		if p.Active {
			n++
		}
	}
	return n
}

// Clone mély másolat (a szinkron ezen dolgozik, és csak siker esetén cseréli le az eredetit).
func (db *B2BDB) Clone() *B2BDB {
	b, _ := json.Marshal(db)
	var c B2BDB
	_ = json.Unmarshal(b, &c)
	return &c
}

func (db *B2BDB) addLog(l ImportLog) ImportLog {
	if n := len(db.Log); n > 0 {
		l.ID = db.Log[n-1].ID + 1
	} else {
		l.ID = 1
	}
	db.Log = append(db.Log, l)
	if len(db.Log) > 60 {
		db.Log = db.Log[len(db.Log)-60:]
	}
	return l
}

// Apply az export alapján frissíti az adatbázist: új → beszúrás, meglévő → frissítés
// (újra feliratkozott → újraaktiválás), kimaradt → inaktiválás (soha nem töröl).
// Ha az export kevesebb partnert ad, mint a jelenlegi aktívak minRatio-szorosa,
// és nincs force, semmi nem változik (ErrSuspiciousExport).
func (db *B2BDB) Apply(list []B2BPartner, rep ExportReport, now time.Time, minRatio float64, force bool) (ImportLog, error) {
	l := ImportLog{Group: db.Group, Start: now, Records: rep.Records, Unique: len(list), Unknown: rep.UnknownKeys, Warnings: rep.Warnings}
	active := db.ActiveCount()
	if len(list) == 0 {
		return l, errors.New("üres export")
	}
	if !force && active > 0 && float64(len(list)) < minRatio*float64(active) {
		return l, fmt.Errorf("%w: az exportban %d partner van, az adatbázisban %d aktív (a küszöb %.0f%%) – a frissítés nem futott le, senki nem lett inaktív",
			ErrSuspiciousExport, len(list), active, minRatio*100)
	}
	idx := make(map[string]*B2BPartner, len(db.Partners))
	for _, p := range db.Partners {
		idx[p.Email] = p
	}
	seen := make(map[string]bool, len(list))
	for i := range list {
		np := list[i]
		seen[np.Email] = true
		if p, ok := idx[np.Email]; ok {
			if !p.sameSource(&np) {
				l.Changed++
			}
			p.copySource(&np)
			if !p.Active {
				l.Reactivated++
				p.Active, p.Inactivated, p.InactiveWhy = true, nil, ""
			} else {
				l.Updated++
			}
			p.LastImport = now
			continue
		}
		np.Active, np.FirstImport, np.LastImport, np.Inactivated, np.InactiveWhy = true, now, now, nil, ""
		db.Partners = append(db.Partners, &np)
		idx[np.Email] = &np
		l.New++
	}
	for _, p := range db.Partners {
		if p.Active && !seen[p.Email] {
			t := now
			p.Active, p.Inactivated, p.InactiveWhy = false, &t, "nincs az exportban (leiratkozott vagy törölt)"
			l.Inactivated++
		}
	}
	sort.Slice(db.Partners, func(i, j int) bool { return db.Partners[i].Email < db.Partners[j].Email })
	l.Active = db.ActiveCount()
	l.OK, l.Result = true, "siker"
	db.SyncedAt = now
	return db.addLog(l), nil
}

// LogFailure a sikertelen futást is naplózza (az adatok nem változnak).
func (db *B2BDB) LogFailure(now time.Time, httpStatus int, reason string, rep *ExportReport) ImportLog {
	l := ImportLog{Group: db.Group, Start: now, HTTP: httpStatus, Result: "megszakítva: " + reason, Active: db.ActiveCount()}
	if rep != nil {
		l.Records, l.Unique, l.Unknown, l.Warnings = rep.Records, rep.Unique, rep.UnknownKeys, rep.Warnings
	}
	return db.addLog(l)
}

// ---------------------------------------------------------------------------
// Partnerhalmaz: szűrés és a szűrők értékei darabszámmal

// PartnerFilter a partnerhalmaz feltételei. Egy szemponton belül VAGY, a szempontok
// között ÉS kapcsolat; a *Not mezővel egy szempont megfordítható („kivéve”).
type PartnerFilter struct {
	Reps        []string `json:"reps,omitempty"` // TK monogram; "" = nincs TK
	RepsNot     bool     `json:"repsNot,omitempty"`
	Levels      []string `json:"levels,omitempty"` // Besor
	LevelsNot   bool     `json:"levelsNot,omitempty"`
	Counties    []string `json:"counties,omitempty"`
	CountiesNot bool     `json:"countiesNot,omitempty"`
	Shops       []string `json:"shops,omitempty"` // Partnerbolt_statusz; "" = nincs adat
	ShopsNot    bool     `json:"shopsNot,omitempty"`
	Props       []string `json:"props,omitempty"` // Tulajdonsag_6; "" = üres
	PropsNot    bool     `json:"propsNot,omitempty"`
	Commission  string   `json:"commission,omitempty"` // "" mind, "only", "exclude"
	Internal    string   `json:"internal,omitempty"`   // Fix (belső másolati címek): "" mind, "only", "exclude"
	SubFrom     string   `json:"subFrom,omitempty"`    // feliratkozás ettől (ÉÉÉÉ-HH-NN)
	SubTo       string   `json:"subTo,omitempty"`      // eddig (a napot is beleértve)
	Query       string   `json:"query,omitempty"`      // név, e-mail, Nazon
	Exclude     []string `json:"exclude,omitempty"`    // egyenként kizárt e-mail címek
}

func inList(list []string, v string, not bool) bool {
	if len(list) == 0 {
		return true
	}
	found := false
	for _, x := range list {
		if x == v {
			found = true
			break
		}
	}
	return found != not
}

func triState(mode string, v bool) bool {
	switch mode {
	case "only":
		return v
	case "exclude":
		return !v
	}
	return true
}

// match: a skip szempontot figyelmen kívül hagyja (a szűrőértékek darabszámához).
func (f *PartnerFilter) match(p *B2BPartner, skip string, excluded map[string]bool) bool {
	if skip != "reps" && !inList(f.Reps, p.RepMono, f.RepsNot) {
		return false
	}
	if skip != "levels" && !inList(f.Levels, p.Level, f.LevelsNot) {
		return false
	}
	if skip != "counties" && !inList(f.Counties, p.County, f.CountiesNot) {
		return false
	}
	if skip != "shops" && !inList(f.Shops, p.Shop, f.ShopsNot) {
		return false
	}
	if skip != "props" && !inList(f.Props, p.Prop6, f.PropsNot) {
		return false
	}
	if skip != "commission" && !triState(f.Commission, p.Commission) {
		return false
	}
	if skip != "internal" && !triState(f.Internal, p.Fix) {
		return false
	}
	if f.SubFrom != "" && (p.Subscribed == "" || p.Subscribed[:10] < f.SubFrom) {
		return false
	}
	if f.SubTo != "" && (p.Subscribed == "" || p.Subscribed[:10] > f.SubTo) {
		return false
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		hay := searchNorm(p.Name + " " + p.Email + " " + p.Nazon + " " + strings.ReplaceAll(p.Email, ".", " "))
		for _, w := range strings.Fields(searchNorm(q)) {
			if !strings.Contains(hay, w) {
				return false
			}
		}
	}
	if skip != "exclude" && excluded[p.Email] {
		return false
	}
	return true
}

func (f *PartnerFilter) excludedSet() map[string]bool {
	m := make(map[string]bool, len(f.Exclude))
	for _, e := range f.Exclude {
		m[strings.ToLower(strings.TrimSpace(e))] = true
	}
	return m
}

// SelectB2B a halmaz partnerei: csak az aktív, levelet kaphatók, a feltételek szerint, név szerint rendezve.
func SelectB2B(db *B2BDB, f PartnerFilter) []*B2BPartner {
	ex := f.excludedSet()
	var out []*B2BPartner
	for _, p := range db.Partners {
		if p.Mailable() && f.match(p, "", ex) {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := searchNorm(out[i].Name), searchNorm(out[j].Name)
		if a != b {
			return a < b
		}
		return out[i].Email < out[j].Email
	})
	return out
}

// FacetValue egy szűrőérték: a jelenlegi többi feltétellel együtt hány partner illeszkedik rá (Count),
// és összesen hány aktív partnernél szerepel (Total).
type FacetValue struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Count int    `json:"count"`
	Total int    `json:"total"`
}

// Facets a szűrőpanel értékei.
type Facets struct {
	Reps       []FacetValue   `json:"reps"`
	Levels     []FacetValue   `json:"levels"`
	Counties   []FacetValue   `json:"counties"`
	Shops      []FacetValue   `json:"shops"`
	Props      []FacetValue   `json:"props"`
	Commission map[string]int `json:"commission"` // yes / no
	Internal   map[string]int `json:"internal"`
}

// B2BLevelOrder a besorolások sorrendje (alacsonytól a magasig).
var B2BLevelOrder = []string{"BASIC", "EZUST", "ARANY", "GYEMANT", "RUBIN", "PLATINA", "TOP", "BIZOMANYOS"}

var b2bLevelLabel = map[string]string{"BASIC": "Basic", "EZUST": "Ezüst", "ARANY": "Arany", "GYEMANT": "Gyémánt",
	"RUBIN": "Rubin", "PLATINA": "Platina", "TOP": "Top", "BIZOMANYOS": "Bizományos"}

// CleanRepName a „ - Energofish Kft.” utótag nélküli képviselőnév.
func CleanRepName(s string) string {
	s = strings.TrimSpace(s)
	for _, suf := range []string{" - Energofish Kft.", " – Energofish Kft.", " - Energofish Kft", " - Energofish"} {
		if strings.HasSuffix(s, suf) {
			return strings.TrimSpace(strings.TrimSuffix(s, suf))
		}
	}
	return s
}

// B2BFacets a szűrőértékek darabszámai (aktív, levelet kapható partnerekre).
func B2BFacets(db *B2BDB, f PartnerFilter) Facets {
	ex := f.excludedSet()
	type acc struct {
		count, total map[string]int
		label        map[string]string
	}
	newAcc := func() *acc { return &acc{map[string]int{}, map[string]int{}, map[string]string{}} }
	reps, levels, counties, shops, props := newAcc(), newAcc(), newAcc(), newAcc(), newAcc()
	fc := Facets{Commission: map[string]int{}, Internal: map[string]int{}}
	for _, p := range db.Partners {
		if !p.Mailable() {
			continue
		}
		add := func(a *acc, key, v, label string) {
			a.total[v]++
			if _, ok := a.label[v]; !ok || a.label[v] == "" {
				a.label[v] = label
			}
			if f.match(p, key, ex) {
				a.count[v]++
			}
		}
		repLabel := "nincs képviselő"
		if p.RepMono != "" || p.RepName != "" {
			repLabel = CleanRepName(p.RepName)
		}
		add(reps, "reps", p.RepMono, repLabel)
		add(levels, "levels", p.Level, b2bLevelLabel[p.Level])
		add(counties, "counties", p.County, p.County)
		add(shops, "shops", p.Shop, map[bool]string{true: "nincs adat", false: p.Shop}[p.Shop == ""])
		add(props, "props", p.Prop6, map[bool]string{true: "(üres)", false: p.Prop6}[p.Prop6 == ""])
		if f.match(p, "commission", ex) {
			fc.Commission[map[bool]string{true: "yes", false: "no"}[p.Commission]]++
		}
		if f.match(p, "internal", ex) {
			fc.Internal[map[bool]string{true: "yes", false: "no"}[p.Fix]]++
		}
	}
	list := func(a *acc, less func(x, y FacetValue) bool) []FacetValue {
		out := []FacetValue{}
		for v, t := range a.total {
			lbl := a.label[v]
			if lbl == "" {
				lbl = v
			}
			out = append(out, FacetValue{Value: v, Label: lbl, Count: a.count[v], Total: t})
		}
		sort.Slice(out, func(i, j int) bool { return less(out[i], out[j]) })
		return out
	}
	byTotal := func(x, y FacetValue) bool {
		if x.Total != y.Total {
			return x.Total > y.Total
		}
		return x.Value < y.Value
	}
	byLabel := func(x, y FacetValue) bool { return searchNorm(x.Label) < searchNorm(y.Label) }
	levelRank := func(v string) int {
		for i, l := range B2BLevelOrder {
			if l == v {
				return i
			}
		}
		return len(B2BLevelOrder)
	}
	fc.Reps = list(reps, func(x, y FacetValue) bool {
		if (x.Value == "") != (y.Value == "") {
			return y.Value == ""
		}
		return byTotal(x, y)
	})
	fc.Levels = list(levels, func(x, y FacetValue) bool {
		if levelRank(x.Value) != levelRank(y.Value) {
			return levelRank(x.Value) < levelRank(y.Value)
		}
		return x.Value < y.Value
	})
	fc.Counties = list(counties, func(x, y FacetValue) bool {
		if (x.Value == "Budapest") != (y.Value == "Budapest") {
			return x.Value == "Budapest"
		}
		return byLabel(x, y)
	})
	fc.Shops = list(shops, byTotal)
	fc.Props = list(props, byTotal)
	return fc
}

// ---------------------------------------------------------------------------
// Átalakítás hírlevél-partnerré

// B2BMapOptions a partnerek levélbe kerülő adatainak beállításai.
type B2BMapOptions struct {
	// Greeting a megszólítás: name (alap) – mindenki a nevével; auto – cégnévnél (csupa nagybetű
	// vagy cégforma) a tartalék megszólítás; fallback – mindenkinek a tartalék megszólítás.
	// A {nev} és {ceg} változó mindhárom esetben ki van töltve.
	Greeting      string            `json:"greeting,omitempty"`
	KeepCaps      bool              `json:"keepCaps,omitempty"`      // a csupa nagybetűs nevek változatlanul (nem „JDB Hungary Zrt.”)
	KeepRepSuffix bool              `json:"keepRepSuffix,omitempty"` // a „ - Energofish Kft.” utótag megtartása a képviselő nevében
	RepPhotos     map[string]string `json:"repPhotos,omitempty"`     // TK monogram → fotó URL
}

// IsCompanyName igaz, ha a név csupa nagybetűs (az exportban így szerepelnek a cégnevek).
func IsCompanyName(s string) bool {
	letters := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsLower(r) {
				return false
			}
		}
	}
	return letters > 1
}

// legalForms cégformák olvasható alakja (a pont és a kötőjel nélküli, kisbetűs alakból).
var legalForms = map[string]string{
	"kft": "Kft.", "bt": "Bt.", "zrt": "Zrt.", "nyrt": "Nyrt.", "rt": "Rt.", "kkt": "Kkt.", "ev": "e.v.",
	"kht": "Kht.", "szov": "Szöv.",
	"sro": "s.r.o.", "as": "a.s.", "gmbh": "GmbH", "ag": "AG", "kg": "KG", "og": "OG", "ug": "UG",
	"srl": "SRL", "sa": "SA", "sl": "S.L.", "doo": "d.o.o.", "ltd": "Ltd.", "llc": "LLC", "spzoo": "Sp. z o.o.",
}

// LooksLikeCompany igaz, ha a név cégnek látszik (csupa nagybetűs, vagy cégformát tartalmaz).
func LooksLikeCompany(s string) bool {
	if IsCompanyName(s) {
		return true
	}
	words := strings.Fields(s)
	for i, w := range words {
		if _, ok := legalForm(w, i, len(words)); ok {
			return true
		}
	}
	return false
}

// legalForm a szó cégforma-alakja, ha a név végén áll vagy pont van benne („Kft.”, „KFT”, „e.v.”).
func legalForm(w string, i, n int) (string, bool) {
	if i == 0 || (i != n-1 && !strings.Contains(w, ".")) {
		return "", false
	}
	lf, ok := legalForms[strings.ToLower(strings.NewReplacer(".", "", "-", "", ",", "").Replace(w))]
	return lf, ok
}

// ReadableName a csupa nagybetűs név olvasható alakja („JDB HUNGARY ZRT.” → „JDB Hungary Zrt.”).
// A vegyes írású neveket nem bántja. A rövid, magánhangzó nélküli szavak (JDB) nagybetűsek maradnak.
func ReadableName(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if !IsCompanyName(s) {
		return s
	}
	words := strings.Split(s, " ")
	for i, w := range words {
		trail := ""
		if strings.HasSuffix(w, ",") {
			trail = ","
		}
		switch lf, ok := legalForm(w, i, len(words)); {
		case ok:
			words[i] = lf + trail
		case i > 0 && w == "ÉS":
			words[i] = "és"
		default:
			words[i] = readableWord(w)
		}
	}
	return strings.Join(words, " ")
}

func readableWord(w string) string {
	letters, vowels, digits := 0, 0, 0
	for _, r := range w {
		switch {
		case unicode.IsLetter(r):
			letters++
			if strings.ContainsRune("AÁEÉIÍOÓÖŐUÚÜŰYaáeéiíoóöőuúüűy", r) {
				vowels++
			}
		case unicode.IsDigit(r):
			digits++
		}
	}
	if letters == 0 || digits > 0 || (letters <= 4 && vowels == 0) {
		return w // rövidítés (JDB, HMS), szám vagy jel
	}
	// minden betűcsoport (kötőjel, perjel, pont után is) nagy kezdőbetűvel
	var b strings.Builder
	start := true
	for _, r := range w {
		if unicode.IsLetter(r) {
			if start {
				b.WriteRune(unicode.ToUpper(r))
			} else {
				b.WriteRune(unicode.ToLower(r))
			}
			start = false
			continue
		}
		b.WriteRune(r)
		start = r == '-' || r == '/' || r == '.' || r == '('
	}
	return b.String()
}

// B2BRegion a „Képviselő területe” mező (a partner megyéje; magyar célcsoportnál „Pest megye”).
func B2BRegion(group, county string) string {
	c := strings.TrimSpace(county)
	if c == "" || group != "B2B_HU" {
		return c
	}
	l := strings.ToLower(c)
	if l == "budapest" || strings.Contains(l, "megye") || strings.Contains(l, "külföld") || strings.Contains(l, "kulfold") {
		return c
	}
	return c + " megye"
}

// B2BTokens a B2B partnertörzsből érkező további változók (a felület változó-listájához).
var B2BTokens = []TokenInfo{
	{"{partnernev}", "Partner neve az exportban szereplő írásmóddal"},
	{"{megye}", "Partner megyéje"},
	{"{nazon}", "Partner azonosítója (Nazon)"},
	{"{besorolas}", "Besorolás (Gyémánt, Arany, Ezüst…)"},
	{"{partnerbolt}", "Partnerbolt-státusz"},
	{"{telefon}", "Partner telefonszáma"},
	{"{feliratkozas}", "Feliratkozás dátuma"},
	{"{bizomanyos}", "Bizományos partner (igen / nem)"},
	{"{tkmonogram}", "Képviselő monogramja"},
	{"{celcsoport}", "Célcsoport (pl. B2B HU)"},
}

// B2BToPartner a hírlevél-partner (az Excel-oszlopokkal azonos mezőkkel). A Nev mező a
// {nev} és a {ceg} változóba is bekerül, a képviselő területe a partner megyéje.
func B2BToPartner(p *B2BPartner, row int, group string, o B2BMapOptions) Partner {
	out := Partner{Row: row, Email: p.Email, RepPhone: p.RepPhone, RepEmail: p.RepEmail,
		Unsubscribe: p.Unsubscribe, Token: p.Token, Source: group}
	name := strings.Join(strings.Fields(p.Name), " ")
	if !o.KeepCaps {
		name = ReadableName(name)
	}
	out.Name, out.Company = name, name
	switch o.Greeting {
	case "fallback":
		out.FallbackGreeting = true
	case "auto":
		out.FallbackGreeting = LooksLikeCompany(p.Name)
	}
	out.RepName = p.RepName
	if !o.KeepRepSuffix {
		out.RepName = CleanRepName(p.RepName)
	}
	if ph := strings.TrimSpace(o.RepPhotos[p.RepMono]); ph != "" {
		out.RepPhoto = ph
	}
	out.RepRegion = B2BRegion(group, p.County)
	extra := map[string]string{
		"nazon": p.Nazon, "megye": p.County, "besorolas": b2bLevelLabel[p.Level], "partnerbolt": p.Shop,
		"telefon": p.Phone, "tkmonogram": p.RepMono, "celcsoport": strings.ReplaceAll(group, "_", " "),
		"bizomanyos": map[bool]string{true: "igen", false: "nem"}[p.Commission], "partnernev": p.Name,
		"feliratkozas": "",
	}
	if extra["besorolas"] == "" {
		extra["besorolas"] = p.Level
	}
	if len(p.Subscribed) >= 10 {
		extra["feliratkozas"] = strings.ReplaceAll(p.Subscribed[:10], "-", ".") + "."
	}
	for k, v := range p.Extra {
		if n := Norm(k); n != "" && extra[n] == "" && !strings.Contains(n, "token") && !strings.Contains(n, "leiratkoz") {
			extra[n] = v
		}
	}
	out.Extra = extra
	return out
}

// B2BSetInfo a betöltött partnerhalmaz adatai (a felületnek és az újraépítéshez).
type B2BSetInfo struct {
	Group    string        `json:"group"`
	Label    string        `json:"label"`
	Name     string        `json:"name,omitempty"` // a mentett halmaz neve
	Summary  string        `json:"summary"`
	Filter   PartnerFilter `json:"filter"`
	SyncedAt time.Time     `json:"syncedAt"`
	Active   int           `json:"active"`   // aktív partnerek a célcsoportban
	Selected int           `json:"selected"` // a halmazban
	NoMail   int           `json:"noMail"`   // aktív, de nem kaphat levelet
}

// FilterSummary a feltételek rövid, olvasható leírása.
func FilterSummary(db *B2BDB, f PartnerFilter) string {
	repName := map[string]string{}
	for _, p := range db.Partners {
		if p.RepMono != "" && repName[p.RepMono] == "" {
			repName[p.RepMono] = CleanRepName(p.RepName)
		}
	}
	var parts []string
	list := func(title string, vals []string, not bool, label func(string) string) {
		if len(vals) == 0 {
			return
		}
		var ls []string
		for _, v := range vals {
			ls = append(ls, label(v))
		}
		pre := ""
		if not {
			pre = "kivéve "
		}
		parts = append(parts, title+": "+pre+strings.Join(ls, ", "))
	}
	list("Képviselő", f.Reps, f.RepsNot, func(v string) string {
		if v == "" {
			return "nincs képviselő"
		}
		if n := repName[v]; n != "" {
			return n
		}
		return v
	})
	list("Besorolás", f.Levels, f.LevelsNot, func(v string) string {
		if l := b2bLevelLabel[v]; l != "" {
			return l
		}
		return v
	})
	list("Megye", f.Counties, f.CountiesNot, func(v string) string { return v })
	list("Bolt", f.Shops, f.ShopsNot, func(v string) string {
		if v == "" {
			return "nincs adat"
		}
		return v
	})
	list("Tulajdonság 6", f.Props, f.PropsNot, func(v string) string {
		if v == "" {
			return "(üres)"
		}
		return v
	})
	switch f.Commission {
	case "only":
		parts = append(parts, "csak bizományosok")
	case "exclude":
		parts = append(parts, "bizományosok nélkül")
	}
	switch f.Internal {
	case "only":
		parts = append(parts, "csak belső másolati címek")
	case "exclude":
		parts = append(parts, "belső másolati címek nélkül")
	}
	if f.SubFrom != "" || f.SubTo != "" {
		parts = append(parts, "feliratkozás: "+strings.TrimSpace(strings.ReplaceAll(f.SubFrom, "-", ".")+" – "+strings.ReplaceAll(f.SubTo, "-", ".")))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		parts = append(parts, "keresés: „"+q+"”")
	}
	if n := len(f.Exclude); n > 0 {
		parts = append(parts, fmt.Sprintf("%d partner egyenként kizárva", n))
	}
	if len(parts) == 0 {
		return "minden aktív partner"
	}
	return strings.Join(parts, " · ")
}
