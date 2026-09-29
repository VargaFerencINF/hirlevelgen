package hirlevel

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	LevelError = "error"
	LevelWarn  = "warn"
	LevelInfo  = "info"

	ScopeContent = "content"
	ScopeProduct = "product"
	ScopePartner = "partner"
	ScopeExcel   = "excel"
	ScopeImage   = "image"
	ScopeOutput  = "output"
)

// Issue egy validálási megállapítás.
type Issue struct {
	Level   string `json:"level"`
	Scope   string `json:"scope"`
	Key     string `json:"key,omitempty"`
	Index   int    `json:"index"`         // termék/partner index (-1, ha nincs)
	Row     int    `json:"row,omitempty"` // Excel-sor
	Message string `json:"message"`
}

var emailRe = regexp.MustCompile(`^[^\s@<>(),;:"\[\]]+@[^\s@<>(),;:"\[\]]+\.[A-Za-z]{2,}$`)

// ValidEmail egyszerű formai ellenőrzés.
func ValidEmail(s string) bool { return emailRe.MatchString(strings.TrimSpace(s)) }

var emailSep = regexp.MustCompile(`[;,\s]+`)

// SplitEmails a cellában lévő (pontosvesszővel, vesszővel vagy szóközzel elválasztott) címek.
func SplitEmails(s string) []string {
	var out []string
	for _, e := range emailSep.Split(strings.TrimSpace(s), -1) {
		if e = strings.TrimSpace(e); e != "" {
			out = append(out, e)
		}
	}
	return out
}

// ValidEmailList igaz, ha a cellában legalább egy cím van, és mind érvényes.
func ValidEmailList(s string) bool {
	list := SplitEmails(s)
	if len(list) == 0 {
		return false
	}
	for _, e := range list {
		if !ValidEmail(e) {
			return false
		}
	}
	return true
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }

// checkURL a link formáját ellenőrzi. kind: "url" vagy "image".
func checkURL(v, kind string) (level, msg string) {
	switch {
	case v == "":
		return "", ""
	case strings.HasPrefix(v, "https://"):
		if strings.ContainsAny(v, " \t\n") {
			return LevelError, "szóközt tartalmaz"
		}
		return "", ""
	case strings.HasPrefix(v, "http://"):
		return LevelWarn, "nem biztonságos http:// cím, használj https://-t"
	case kind == "url" && (strings.HasPrefix(v, "mailto:") || strings.HasPrefix(v, "tel:")):
		return "", ""
	default:
		return LevelError, "https://, tel: vagy mailto: kezdetű teljes cím kell"
	}
}

// IsLocalAsset igaz, ha a kép a helyi (feltöltetlen) képtárra mutat.
func IsLocalAsset(v string) bool {
	return strings.HasPrefix(v, "{assets}") || strings.HasPrefix(v, "../assets") || strings.HasPrefix(v, "/assets") || strings.HasPrefix(v, "assets/")
}

// ValidateContent a közös tartalmat ellenőrzi az adott sablonhoz.
func ValidateContent(c Content, products []Product, tpl *Template) []Issue {
	var out []Issue
	add := func(level, key, msg string) {
		out = append(out, Issue{Level: level, Scope: ScopeContent, Key: key, Index: -1, Message: msg})
	}
	short := ""
	if tpl != nil {
		short = tpl.Short
	}
	sample := PartnerTokens(&SamplePartner)
	sample["termekszam"] = fmt.Sprint(len(SelectedProducts(products)))
	sample["assets"] = "https://pelda.hu/assets"
	sample["cikkszam"] = "T00000"

	pollOn := tpl != nil && tpl.HasPoll && strings.TrimSpace(c["poll.question"]) != ""
	assetsEmpty := strings.TrimSpace(c["assets.base"]) == ""

	for i := range Fields {
		f := &Fields[i]
		if short != "" && !f.UsedIn(short) {
			continue
		}
		v := strings.TrimSpace(c[f.Key])
		isPoll := strings.HasPrefix(f.Key, "poll.") || f.Key == "cover.meta.poll"
		if isPoll && !pollOn {
			continue
		}
		if v == "" {
			if f.Required || (isPoll && (f.Key == "poll.label" || f.Key == "cover.meta.poll")) {
				add(LevelError, f.Key, fmt.Sprintf("Kötelező mező üres: %s", f.Label))
			}
			continue
		}
		ev, unknown, _ := Expand(v, sample, f.IsURLKind())
		for _, u := range unknown {
			if f.Key == "offer.imagePattern" || f.Key == "offer.urlPattern" {
				continue
			}
			add(LevelWarn, f.Key, fmt.Sprintf("%s: ismeretlen változó %s (elírás?)", f.Label, u))
		}
		if f.IsURLKind() {
			if f.Kind == "image" && IsLocalAsset(v) {
				continue // a képtár címe külön ellenőrizve
			}
			if lvl, msg := checkURL(ev, f.Kind); lvl != "" {
				if f.Kind == "image" && lvl == LevelError && !strings.Contains(v, "://") {
					msg = "teljes https:// kezdetű kép cím kell (vagy {assets}/… a képtárból)"
				}
				add(lvl, f.Key, fmt.Sprintf("%s: %s", f.Label, msg))
			}
			if f.Key == "assets.base" && strings.HasSuffix(v, "/") {
				add(LevelInfo, f.Key, "A képtár címének végén lévő perjelet a program elhagyja.")
			}
			continue
		}
		n := runeLen(strings.TrimSpace(ev))
		if f.Soft > 0 && n > f.Soft {
			add(LevelWarn, f.Key, fmt.Sprintf("%s: %d karakter, javasolt legfeljebb %d", f.Label, n, f.Soft))
		}
		if f.Min > 0 && n < f.Min {
			add(LevelInfo, f.Key, fmt.Sprintf("%s: %d karakter, javasolt legalább %d", f.Label, n, f.Min))
		}
		if f.MaxWords > 0 {
			if w := len(strings.Fields(ev)); w > f.MaxWords {
				add(LevelWarn, f.Key, fmt.Sprintf("%s: %d szó, javasolt 3-%d", f.Label, w, f.MaxWords))
			}
		}
	}
	if strings.Contains(c["footer.address"], "[") && strings.Contains(c["footer.address"], "]") {
		add(LevelWarn, "footer.address", "A lábléc címe még a sablon helyőrzőit tartalmazza ([Székhely címe] stb.).")
	}
	if assetsEmpty {
		add(LevelWarn, "assets.base", "A képtár (assets) webcíme üres: a logó, a hullámok és a {assets}/… képek csak helyi megtekintésnél látszanak. Kiküldés előtt töltsd fel az assets mappát, és add meg a címét.")
	}
	if pollOn {
		answers := 0
		for i := 1; i <= 3; i++ {
			t := strings.TrimSpace(c[fmt.Sprintf("poll.answers.%d", i)])
			u := strings.TrimSpace(c[fmt.Sprintf("poll.answers.%d.url", i)])
			switch {
			case t != "" && u == "":
				add(LevelError, fmt.Sprintf("poll.answers.%d.url", i), fmt.Sprintf("A(z) %d. válasznak nincs linkje.", i))
			case t == "" && u != "":
				add(LevelWarn, fmt.Sprintf("poll.answers.%d", i), fmt.Sprintf("A(z) %d. válasznak van linkje, de nincs felirata, így nem jelenik meg.", i))
			case t != "":
				answers++
			}
		}
		if answers < 2 {
			add(LevelError, "poll.answers.1", "A kérdéshez legalább 2 válasz kell (vagy töröld a kérdést a blokk elhagyásához).")
		}
	}
	return out
}

// ValidateProducts a termékeket ellenőrzi.
func ValidateProducts(c Content, products []Product) []Issue {
	var out []Issue
	add := func(level string, idx int, key, msg string) {
		out = append(out, Issue{Level: level, Scope: ScopeProduct, Key: key, Index: idx, Message: msg})
	}
	sel := 0
	codes := map[string]int{}
	tokens := PartnerTokens(&SamplePartner)
	tokens["assets"] = "https://pelda.hu/assets"
	for i, p := range products {
		if !p.On {
			continue
		}
		sel++
		label := fmt.Sprintf("%d. termék", i+1)
		if p.Code != "" {
			label += " (" + p.Code + ")"
		}
		if strings.TrimSpace(p.Code) == "" {
			add(LevelError, i, "code", label+": hiányzik a cikkszám")
		} else if prev, ok := codes[strings.ToUpper(p.Code)]; ok {
			add(LevelWarn, i, "code", fmt.Sprintf("%s: a cikkszám már szerepel a(z) %d. terméknél", label, prev+1))
		} else {
			codes[strings.ToUpper(p.Code)] = i
		}
		if strings.TrimSpace(p.Name) == "" {
			add(LevelError, i, "name", label+": hiányzik a cikknév")
		}
		if n := runeLen(strings.TrimSpace(p.Name)); n > 26 {
			add(LevelWarn, i, "name", fmt.Sprintf("%s: a név %d karakter, max. kb. 26 fér ki 2 sorban", label, n))
		}
		if n := runeLen(strings.TrimSpace(p.Desc)); n > 30 {
			add(LevelWarn, i, "desc", fmt.Sprintf("%s: a leírás %d karakter, max. kb. 30 fér ki 2 sorban", label, n))
		}
		if n := runeLen(strings.TrimSpace(p.Deal)); n > 18 {
			add(LevelWarn, i, "deal", fmt.Sprintf("%s: az akció szövege %d karakter, max. kb. 18 fér ki 1 sorban", label, n))
		}
		it := BuildItem(c, p, tokens, "")
		if it["url"] == "" {
			add(LevelError, i, "url", label+": nincs gomb link (és alapértelmezett minta sincs megadva)")
		} else if lvl, msg := checkURL(it["url"], "url"); lvl != "" {
			add(lvl, i, "url", label+": gomb link: "+msg)
		}
		if it["image"] == "" {
			add(LevelError, i, "image", label+": nincs kép link (és alapértelmezett minta sincs megadva)")
		} else if !IsLocalAsset(p.Image) {
			if lvl, msg := checkURL(it["image"], "image"); lvl != "" {
				add(lvl, i, "image", label+": kép link: "+msg)
			}
		}
	}
	switch {
	case sel == 0:
		add(LevelError, -1, "", "Nincs kiválasztott termék: tölts be terméklistát az Excelből, vagy vegyél fel terméket.")
	case sel > 12:
		add(LevelWarn, -1, "", fmt.Sprintf("%d termék van kiválasztva: ez hosszú levél lesz (javasolt 3, 6 vagy 9).", sel))
	case sel%GridColumns != 0:
		add(LevelInfo, -1, "", fmt.Sprintf("A rács asztali nézetben 3 oszlopos: %d termékkel az utolsó sor hiányos lesz (a 3, 6, 9 mutat a legjobban).", sel))
	}
	return out
}

// ValidPhoto igaz, ha a képviselő kép linkje használható.
func ValidPhoto(raw string) bool {
	v := strings.TrimSpace(raw)
	return strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "http://") || IsLocalAsset(v)
}

// ValidatePartners a partnerlistát ellenőrzi.
func ValidatePartners(partners []Partner) []Issue {
	var out []Issue
	seen := map[string]int{}
	for i := range partners {
		p := &partners[i]
		add := func(level, key, msg string) {
			out = append(out, Issue{Level: level, Scope: ScopePartner, Key: key, Index: i, Row: p.Row, Message: msg})
		}
		who := p.Email
		if who == "" {
			who = p.Name
		}
		prefix := fmt.Sprintf("%d. sor (%s): ", p.Row, who)
		switch {
		case p.Email == "":
			add(LevelError, "email", prefix+"hiányzik az e-mail cím")
		case !ValidEmailList(p.Email):
			add(LevelError, "email", prefix+"érvénytelen e-mail cím")
		default:
			k := strings.ToLower(p.Email)
			if prev, ok := seen[k]; ok {
				add(LevelWarn, "email", fmt.Sprintf("%saz e-mail cím már szerepel a(z) %d. sorban", prefix, partners[prev].Row))
			} else {
				seen[k] = i
			}
		}
		if strings.TrimSpace(p.Name) == "" {
			add(LevelInfo, "name", prefix+"nincs név, a tartalék megszólítás kerül a levélbe")
		}
		if strings.TrimSpace(p.RepName) == "" {
			add(LevelWarn, "repName", prefix+"nincs területi képviselő")
		}
		if ph := strings.TrimSpace(p.RepPhoto); ph != "" && !ValidPhoto(ph) {
			add(LevelWarn, "repPhoto", prefix+"a képviselő kép linkje nem https:// kezdetű, helyette monogram jelenik meg")
		}
		if strings.TrimSpace(p.RepPhone) != "" && len(strings.TrimPrefix(PhoneURL(p.RepPhone), "tel:+")) < 9 {
			add(LevelWarn, "repPhone", prefix+"a képviselő telefonszáma túl rövidnek tűnik")
		}
		if e := strings.TrimSpace(p.RepEmail); e != "" && !ValidEmail(e) {
			add(LevelWarn, "repEmail", prefix+"a képviselő e-mail címe érvénytelen, a gomb elmarad")
		}
	}
	return out
}

// PartnerBlocked igaz, ha a partnernek van hibája (így kimarad a generálásból).
func PartnerBlocked(p *Partner) bool {
	return p.Email == "" || !ValidEmailList(p.Email)
}

// ValidateColumns a partnerlista felülíró oszlopait ellenőrzi.
func ValidateColumns(ex *ExcelData, tpls []*Template) []Issue {
	if ex == nil {
		return nil
	}
	known := map[string]bool{}
	for _, t := range tpls {
		for _, k := range t.Keys {
			known[k] = true
		}
	}
	for _, f := range Fields {
		known[f.Key] = true
	}
	var out []Issue
	for _, c := range ex.PartnerColumns {
		if c.Field == "override" && !known[c.Target] {
			out = append(out, Issue{Level: LevelWarn, Scope: ScopeExcel, Index: -1,
				Message: fmt.Sprintf("A(z) „%s” oszlop sablonkulcsnak tűnik, de egyik sablon sem használja.", c.Header)})
		}
	}
	return out
}

// CountIssues szintenként megszámolja a megállapításokat.
func CountIssues(list []Issue) map[string]int {
	m := map[string]int{LevelError: 0, LevelWarn: 0, LevelInfo: 0}
	for _, i := range list {
		m[i.Level]++
	}
	return m
}
