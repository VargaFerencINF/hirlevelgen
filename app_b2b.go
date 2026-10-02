package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	h "energofish/hirlevel/internal/hirlevel"
)

// B2B partnertörzs (Webgalamb export) mint partnerforrás. A forrás-tokenek titkosak:
// a felület csak kitakart alakjukat látja, naplóba és hibaüzenetbe nem kerülnek.

// B2BSettings a nem titkos beállítások (beallitasok.json).
type B2BSettings struct {
	Group   string          `json:"group,omitempty"`
	Filter  h.PartnerFilter `json:"filter"`
	Options h.B2BMapOptions `json:"options"`
	Presets []B2BPreset     `json:"presets,omitempty"`
	Loaded  *B2BLoaded      `json:"loaded,omitempty"` // a hírlevélhez betöltött halmaz (újraindításkor visszaépül)
}

// B2BPreset egy elmentett partnerhalmaz.
type B2BPreset struct {
	Name   string          `json:"name"`
	Group  string          `json:"group"`
	Filter h.PartnerFilter `json:"filter"`
}

// B2BLoaded a betöltött halmaz meghatározása.
type B2BLoaded struct {
	Group  string          `json:"group"`
	Name   string          `json:"name,omitempty"`
	Filter h.PartnerFilter `json:"filter"`
}

const sourcesMagic = "EFSRC1\n"

func (a *App) sourcesFile() string {
	if a.configDir == "" {
		return ""
	}
	return filepath.Join(a.configDir, "partnerforrasok.dat")
}

// loadSources a mentett forráscímek (titkosítva tárolva).
func (a *App) loadSources() map[string]string {
	out := map[string]string{}
	path := a.sourcesFile()
	if path == "" {
		return out
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	if bytes.HasPrefix(data, []byte(sourcesMagic)) {
		if unprotectSecret == nil {
			return out
		}
		if data, err = unprotectSecret(data[len(sourcesMagic):]); err != nil {
			log.Printf("a partnerforrások nem fejthetők vissza ezen a gépen/felhasználóval")
			return map[string]string{}
		}
	}
	_ = json.Unmarshal(data, &out)
	return out
}

func (a *App) saveSources(m map[string]string) error {
	path := a.sourcesFile()
	if path == "" {
		return nil
	}
	data, _ := json.Marshal(m)
	if protectSecret != nil {
		enc, err := protectSecret(data)
		if err != nil {
			return fmt.Errorf("a forrás titkosítása nem sikerült: %v", err)
		}
		data = append([]byte(sourcesMagic), enc...)
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// sourceFor a célcsoport forráscíme: környezeti változó (WEBGALAMB_TOKEN_B2B_HU…) vagy a mentett.
func (a *App) sourceFor(group string) (src, origin string) {
	if g := h.FindB2BGroup(group); g != nil {
		if v := h.NormalizeB2BSource(os.Getenv(g.Env)); v != "" {
			return v, "env"
		}
	}
	if v := a.loadSources()[group]; v != "" {
		return v, "saved"
	}
	return "", ""
}

func validGroup(g string) (string, error) {
	if gr := h.FindB2BGroup(g); gr != nil {
		return gr.ID, nil
	}
	return "", errors.New("ismeretlen célcsoport")
}

type b2bGroupView struct {
	h.B2BGroup
	Configured bool         `json:"configured"`
	Origin     string       `json:"origin,omitempty"` // env, saved
	Masked     string       `json:"masked,omitempty"`
	SyncedAt   string       `json:"syncedAt,omitempty"`
	Active     int          `json:"active"`
	Mailable   int          `json:"mailable"`
	Inactive   int          `json:"inactive"`
	Total      int          `json:"total"`
	LastLog    *h.ImportLog `json:"lastLog,omitempty"`
	Error      string       `json:"dbError,omitempty"`
}

func (a *App) b2bState() map[string]any {
	groups := make([]b2bGroupView, 0, len(h.B2BGroups))
	for _, g := range h.B2BGroups {
		v := b2bGroupView{B2BGroup: g}
		if src, origin := a.sourceFor(g.ID); src != "" {
			v.Configured, v.Origin, v.Masked = true, origin, h.MaskSource(src)
		}
		db, err := a.b2b.DB(g.ID)
		if err != nil {
			v.Error = err.Error()
		} else {
			v.Total = len(db.Partners)
			for _, p := range db.Partners {
				switch {
				case p.Mailable():
					v.Mailable++
					v.Active++
				case p.Active:
					v.Active++
				default:
					v.Inactive++
				}
			}
			if !db.SyncedAt.IsZero() {
				v.SyncedAt = db.SyncedAt.Format(time.RFC3339)
			}
			if n := len(db.Log); n > 0 {
				l := db.Log[n-1]
				v.LastLog = &l
			}
		}
		groups = append(groups, v)
	}
	a.mu.Lock()
	set := a.b2bSet
	a.mu.Unlock()
	if set.Group == "" {
		set.Group = "B2B_HU"
	}
	return map[string]any{"groups": groups, "settings": set, "protected": secretsProtected}
}

func (a *App) apiB2BState(w http.ResponseWriter, r *http.Request) (any, error) {
	return a.b2bState(), nil
}

// apiB2BSources a forráscímek mentése: beillesztett szövegből (több célcsoport) vagy egyenként.
func (a *App) apiB2BSources(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Text   string `json:"text"`
		Group  string `json:"group"`
		Value  string `json:"value"`
		Remove bool   `json:"remove"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	m := a.loadSources()
	var saved []string
	switch {
	case req.Remove:
		g, err := validGroup(req.Group)
		if err != nil {
			return nil, err
		}
		delete(m, g)
	case strings.TrimSpace(req.Text) != "":
		found := h.ParseB2BSources(req.Text)
		if len(found) == 0 {
			return nil, errors.New("nem találtam célcsoportot és tokenes linket (minta: „B2B HU: https://…&token=…”)")
		}
		for g, u := range found {
			m[g] = u
			saved = append(saved, strings.ReplaceAll(g, "_", " "))
		}
	default:
		g, err := validGroup(req.Group)
		if err != nil {
			return nil, err
		}
		u := h.NormalizeB2BSource(req.Value)
		if u == "" {
			return nil, errors.New("a megadott érték nem token és nem tokenes link")
		}
		m[g] = u
		saved = append(saved, strings.ReplaceAll(g, "_", " "))
	}
	if err := a.saveSources(m); err != nil {
		return nil, err
	}
	sort.Strings(saved)
	st := a.b2bState()
	st["saved"] = saved
	return st, nil
}

// apiB2BSync letölti az exportot és frissíti a helyi partnertörzset.
func (a *App) apiB2BSync(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Group string `json:"group"`
		Force bool   `json:"force"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	g, err := validGroup(req.Group)
	if err != nil {
		return nil, err
	}
	res, err := a.syncGroup(g, req.Force)
	out := a.b2bState()
	out["result"] = res
	if err != nil {
		// nem „error” kulcson: a felület felajánlhatja a kényszerített futtatást
		out["failed"] = h.MaskSecrets(err.Error())
		out["suspicious"] = errors.Is(err, h.ErrSuspiciousExport)
	}
	return out, nil
}

func (a *App) syncGroup(group string, force bool) (h.SyncResult, error) {
	src, _ := a.sourceFor(group)
	if src == "" {
		return h.SyncResult{}, errors.New("ehhez a célcsoporthoz nincs megadva forrás (token) – Partnertörzs › Források")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	res, err := a.b2b.Sync(ctx, group, src, force)
	l := res.Log
	if err != nil {
		log.Printf("partnertörzs %s: szinkron megszakítva (%s)", group, h.MaskSecrets(err.Error()))
	} else {
		log.Printf("partnertörzs %s: %d rekord, %d új, %d frissítve (%d változott), %d újraaktivált, %d inaktivált, %d aktív",
			group, l.Records, l.New, l.Updated, l.Changed, l.Reactivated, l.Inactivated, l.Active)
	}
	return res, err
}

// b2bRow egy partner a halmazválasztóban (token és leiratkozó link nélkül).
type b2bRow struct {
	Email       string `json:"email"`
	Name        string `json:"name"`
	Nazon       string `json:"nazon"`
	RepMono     string `json:"repMono"`
	Rep         string `json:"rep"`
	Level       string `json:"level"`
	County      string `json:"county"`
	Shop        string `json:"shop"`
	Prop6       string `json:"prop6"`
	Subscribed  string `json:"subscribed"`
	Fix         bool   `json:"fix"`
	Commission  bool   `json:"commission"`
	NoToken     bool   `json:"noToken"`
	NoMail      string `json:"noMail,omitempty"`
	Excluded    bool   `json:"excluded,omitempty"`
	Inactivated string `json:"inactivated,omitempty"`
	InactiveWhy string `json:"inactiveWhy,omitempty"`
}

func toRow(p *h.B2BPartner) b2bRow {
	r := b2bRow{Email: p.Email, Name: p.Name, Nazon: p.Nazon, RepMono: p.RepMono, Rep: h.CleanRepName(p.RepName), Level: p.Level,
		County: p.County, Shop: p.Shop, Prop6: p.Prop6, Subscribed: p.Subscribed, Fix: p.Fix, Commission: p.Commission,
		NoToken: p.Token == "", NoMail: p.NoMail, InactiveWhy: p.InactiveWhy}
	if p.Inactivated != nil {
		r.Inactivated = p.Inactivated.Format(time.RFC3339)
	}
	return r
}

// apiB2BQuery a szűrők értékei darabszámokkal és a halmaz (első N) partnere.
func (a *App) apiB2BQuery(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Group   string           `json:"group"`
		Filter  h.PartnerFilter  `json:"filter"`
		Options *h.B2BMapOptions `json:"options"`
		Limit   int              `json:"limit"`
		List    string           `json:"list"` // "" = a halmaz, "excluded", "inactive", "nomail"
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	g, err := validGroup(req.Group)
	if err != nil {
		return nil, err
	}
	db, err := a.b2b.DB(g)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.b2bSet.Group, a.b2bSet.Filter = g, req.Filter
	if req.Options != nil {
		a.b2bSet.Options = *req.Options
	}
	a.scheduleSave()
	a.mu.Unlock()

	if req.Limit <= 0 || req.Limit > 2000 {
		req.Limit = 300
	}
	sel := h.SelectB2B(db, req.Filter)
	var rows []b2bRow
	excluded := map[string]bool{}
	for _, e := range req.Filter.Exclude {
		excluded[strings.ToLower(e)] = true
	}
	inactive, noMail := 0, 0
	for _, p := range db.Partners {
		if !p.Active {
			inactive++
		} else if p.NoMail != "" {
			noMail++
		}
	}
	switch req.List {
	case "inactive":
		var list []*h.B2BPartner
		for _, p := range db.Partners {
			if !p.Active {
				list = append(list, p)
			}
		}
		sort.Slice(list, func(i, j int) bool {
			if list[i].Inactivated != nil && list[j].Inactivated != nil && !list[i].Inactivated.Equal(*list[j].Inactivated) {
				return list[i].Inactivated.After(*list[j].Inactivated)
			}
			return list[i].Email < list[j].Email
		})
		for _, p := range list {
			rows = append(rows, toRow(p))
		}
	case "nomail":
		for _, p := range db.Partners {
			if p.Active && p.NoMail != "" {
				rows = append(rows, toRow(p))
			}
		}
	case "excluded":
		for _, p := range db.Partners {
			if p.Mailable() && excluded[p.Email] {
				r := toRow(p)
				r.Excluded = true
				rows = append(rows, r)
			}
		}
	default:
		for _, p := range sel {
			rows = append(rows, toRow(p))
		}
	}
	more := len(rows) > req.Limit
	if more {
		rows = rows[:req.Limit]
	}
	if rows == nil {
		rows = []b2bRow{}
	}
	reps := map[string]bool{}
	for _, p := range sel {
		reps[p.RepMono] = true
	}
	mailable := 0
	for _, p := range db.Partners {
		if p.Mailable() {
			mailable++
		}
	}
	return map[string]any{
		"facets": h.B2BFacets(db, req.Filter), "count": len(sel), "mailable": mailable, "inactive": inactive, "noMail": noMail,
		"excluded": len(excluded), "reps": len(reps), "rows": rows, "more": more, "summary": h.FilterSummary(db, req.Filter),
		"synced": !db.SyncedAt.IsZero(),
	}, nil
}

// apiB2BReps a célcsoport képviselői (a fotók megadásához).
func (a *App) apiB2BReps(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Group string `json:"group"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	g, err := validGroup(req.Group)
	if err != nil {
		return nil, err
	}
	db, err := a.b2b.DB(g)
	if err != nil {
		return nil, err
	}
	type rep struct {
		Mono     string `json:"mono"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Partners int    `json:"partners"`
		Photo    string `json:"photo"`
	}
	a.mu.Lock()
	photos := a.b2bSet.Options.RepPhotos
	a.mu.Unlock()
	m := map[string]*rep{}
	for _, p := range db.Partners {
		if !p.Mailable() || p.RepMono == "" {
			continue
		}
		x := m[p.RepMono]
		if x == nil {
			x = &rep{Mono: p.RepMono, Name: h.CleanRepName(p.RepName), Email: p.RepEmail, Phone: p.RepPhone, Photo: photos[p.RepMono]}
			m[p.RepMono] = x
		}
		x.Partners++
	}
	out := []*rep{}
	for _, x := range m {
		out = append(out, x)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Partners != out[j].Partners {
			return out[i].Partners > out[j].Partners
		}
		return out[i].Mono < out[j].Mono
	})
	return map[string]any{"reps": out}, nil
}

// apiB2BOptions a levélbe kerülő adatok beállításai (megszólítás, utótag, képviselő-fotók).
func (a *App) apiB2BOptions(w http.ResponseWriter, r *http.Request) (any, error) {
	var o h.B2BMapOptions
	if err := decode(r, &o); err != nil {
		return nil, err
	}
	for k, v := range o.RepPhotos {
		if v = strings.TrimSpace(v); v == "" {
			delete(o.RepPhotos, k)
		} else {
			o.RepPhotos[k] = v
		}
	}
	a.mu.Lock()
	a.b2bSet.Options = o
	loaded := a.b2bSet.Loaded
	a.scheduleSave()
	a.mu.Unlock()
	// a betöltött halmaz azonnal az új beállításokkal (pl. új képviselő-fotó)
	if loaded != nil && a.excelIsB2B() {
		if err := a.loadB2BSet(loaded.Group, loaded.Filter, loaded.Name, nil); err != nil {
			return nil, err
		}
		return a.excelResponse()
	}
	return map[string]any{"ok": true}, nil
}

func (a *App) excelIsB2B() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.excel != nil && a.excel.Source == "b2b"
}

// apiB2BPresets a mentett partnerhalmazok kezelése.
func (a *App) apiB2BPresets(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Action string          `json:"action"` // save, delete
		Name   string          `json:"name"`
		Group  string          `json:"group"`
		Filter h.PartnerFilter `json:"filter"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("adj nevet a halmaznak")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	list := a.b2bSet.Presets[:0:0]
	for _, p := range a.b2bSet.Presets {
		if !strings.EqualFold(p.Name, name) {
			list = append(list, p)
		}
	}
	if req.Action == "save" {
		g, err := validGroup(req.Group)
		if err != nil {
			return nil, err
		}
		list = append(list, B2BPreset{Name: name, Group: g, Filter: req.Filter})
		sort.Slice(list, func(i, j int) bool { return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name) })
	}
	a.b2bSet.Presets = list
	a.scheduleSave()
	return map[string]any{"presets": list}, nil
}

// apiB2BLoad a halmazt betölti partnerlistaként (az Excel helyére).
func (a *App) apiB2BLoad(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Group   string           `json:"group"`
		Filter  h.PartnerFilter  `json:"filter"`
		Name    string           `json:"name"`
		Options *h.B2BMapOptions `json:"options"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	g, err := validGroup(req.Group)
	if err != nil {
		return nil, err
	}
	if err := a.loadB2BSet(g, req.Filter, strings.TrimSpace(req.Name), req.Options); err != nil {
		return nil, err
	}
	return a.excelResponse()
}

// buildB2BData a halmazból Excel-szerű partnerlista (a program többi része változatlanul használja).
func (a *App) buildB2BData(group string, f h.PartnerFilter, name string, o h.B2BMapOptions) (*h.ExcelData, error) {
	db, err := a.b2b.DB(group)
	if err != nil {
		return nil, err
	}
	if db.SyncedAt.IsZero() {
		return nil, errors.New("ehhez a célcsoporthoz még nem volt szinkron – előbb töltsd le a partnertörzset")
	}
	sel := h.SelectB2B(db, f)
	if len(sel) == 0 {
		return nil, errors.New("a halmaz üres – lazíts a feltételeken")
	}
	label := strings.ReplaceAll(group, "_", " ")
	if gr := h.FindB2BGroup(group); gr != nil {
		label = gr.Label
	}
	ex := &h.ExcelData{FileName: label + " partnertörzs", LoadedAt: time.Now(), Source: "b2b", PartnerSheet: label}
	if name != "" {
		ex.FileName = name + " · " + label
	}
	for i, p := range sel {
		ex.Partners = append(ex.Partners, h.B2BToPartner(p, i+1, group, o))
	}
	noMail := 0
	for _, p := range db.Partners {
		if p.Active && p.NoMail != "" {
			noMail++
		}
	}
	ex.B2B = &h.B2BSetInfo{Group: group, Label: label, Name: name, Summary: h.FilterSummary(db, f), Filter: f,
		SyncedAt: db.SyncedAt, Active: db.ActiveCount(), Selected: len(sel), NoMail: noMail}
	col := func(hdr, field, label, target string) h.Column {
		return h.Column{Index: len(ex.PartnerColumns), Letter: "·", Header: hdr, Field: field, Label: label, Target: target}
	}
	greet := map[string]string{"": "Partner neve (csupa nagybetűs cégnévnél tartalék megszólítás)", "auto": "Partner neve (csupa nagybetűs cégnévnél tartalék megszólítás)",
		"name": "Partner neve (megszólítás)", "fallback": "Cégnév (mindenkinek a tartalék megszólítás)"}[o.Greeting]
	ex.PartnerColumns = []h.Column{
		col("Email_cim", "email", "Partner e-mail", ""),
		col("Nev", "name", greet, ""),
		col("Teruleti_kepviselo_nev", "repName", "Területi képviselő neve", ""),
		col("Teruleti_kepviselo_telefonszam", "repPhone", "Képviselő telefon", ""),
		col("Teruleti_kepviselo_email_cim", "repEmail", "Képviselő e-mail", ""),
		col("Teruleti_kepviselo_monogram", "repPhoto", "Képviselő fotó (a monogramhoz megadott kép)", ""),
		col("Leiratkozas_link", "override", "Leiratkozás linkje – partnerenként (soha nem nyitjuk meg)", "footer.unsubscribe.url"),
		col("Nazon", "extra", "Változó: {nazon}", "nazon"),
		col("Megye", "extra", "Változó: {megye}", "megye"),
		col("Besor", "extra", "Változó: {besorolas}", "besorolas"),
		col("Partnerbolt_statusz", "extra", "Változó: {partnerbolt}", "partnerbolt"),
		col("Telefonszam", "extra", "Változó: {telefon}", "telefon"),
		col("Feliratkozas_datum", "extra", "Változó: {feliratkozas}", "feliratkozas"),
		col("Token", "", "Titkos – tárolva a későbbi link-kiegészítéshez", ""),
	}
	ex.Issues = []h.Issue{}
	return ex, nil
}

func (a *App) loadB2BSet(group string, f h.PartnerFilter, name string, opts *h.B2BMapOptions) error {
	a.mu.Lock()
	if opts != nil {
		a.b2bSet.Options = *opts
	}
	o := a.b2bSet.Options
	a.mu.Unlock()
	ex, err := a.buildB2BData(group, f, name, o)
	if err != nil {
		return err
	}
	issues := h.ValidatePartners(ex.Partners)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.excel, a.excelIssue = ex, issues
	a.b2bSet.Group, a.b2bSet.Filter = group, f
	a.b2bSet.Loaded = &B2BLoaded{Group: group, Name: name, Filter: f}
	a.scheduleSave()
	return nil
}

// refreshB2BBeforeSend a küldés előtti kötelező frissítés (S10): szinkron, majd a halmaz újraépítése.
// A kiválasztott partnereket e-mail alapján követi; az új, a feltételeknek megfelelő partnerek bekerülnek,
// a leiratkozottak kimaradnak.
func (a *App) refreshB2BBeforeSend(only []int) ([]int, map[string]any, error) {
	a.mu.Lock()
	loaded := a.b2bSet.Loaded
	var oldEmails []string
	selected := map[string]bool{}
	if a.excel != nil {
		for _, p := range a.excel.Partners {
			oldEmails = append(oldEmails, p.Email)
		}
		for _, i := range only {
			if i >= 0 && i < len(a.excel.Partners) {
				selected[a.excel.Partners[i].Email] = true
			}
		}
	}
	a.mu.Unlock()
	if loaded == nil {
		return only, nil, nil
	}
	res, err := a.syncGroup(loaded.Group, false)
	if err != nil {
		return nil, map[string]any{"syncFailed": h.MaskSecrets(err.Error()), "suspicious": errors.Is(err, h.ErrSuspiciousExport)}, err
	}
	if err := a.loadB2BSet(loaded.Group, loaded.Filter, loaded.Name, nil); err != nil {
		return nil, nil, err
	}
	old := map[string]bool{}
	for _, e := range oldEmails {
		old[e] = true
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	var next []int
	added, dropped := 0, 0
	now := map[string]bool{}
	for i, p := range a.excel.Partners {
		now[p.Email] = true
		switch {
		case selected[p.Email]:
			next = append(next, i)
		case !old[p.Email]:
			next = append(next, i)
			added++
		}
	}
	for e := range selected {
		if !now[e] {
			dropped++
		}
	}
	info := map[string]any{"log": res.Log, "added": added, "dropped": dropped, "only": next}
	return next, info, nil
}
