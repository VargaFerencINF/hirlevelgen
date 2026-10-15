package hirlevel

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// SelectedProducts a bekapcsolt termékek.
func SelectedProducts(products []Product) []Product {
	out := make([]Product, 0, min(len(products), 16))
	for _, p := range products {
		if p.On {
			out = append(out, p)
		}
	}
	return out
}

// MaxNewsletterProducts egy levélbe kerülő termékek felső korlátja (fölötte hiba; az előnézet
// és a levél is legfeljebb ennyit mutat, hogy egy hibás terméklista ne bénítsa meg a programot).
const MaxNewsletterProducts = 48

// ResolveAssets az assets.base végső értéke; üres mezőnél a fallback (relatív út).
func ResolveAssets(c Content, fallback string) string {
	a := strings.TrimRight(strings.TrimSpace(c["assets.base"]), "/")
	if a == "" {
		return fallback
	}
	return a
}

// Build összeállítja egy partner hírlevelének kitöltési adatait.
// assetsFallback: a képtár relatív címe, ha az assets.base üres (pl. "../assets").
func Build(c Content, products []Product, p *Partner, tpl *Template, assetsFallback string) *RenderData {
	sel := SelectedProducts(products)
	if len(sel) > MaxNewsletterProducts {
		sel = sel[:MaxNewsletterProducts]
	}
	assets := ResolveAssets(c, assetsFallback)

	tokens := PartnerTokens(p)
	tokens["termekszam"] = strconv.Itoa(len(sel))
	tokens["assets"] = assets

	values := make(map[string]string, len(Fields)+32)
	for i := range Fields {
		f := &Fields[i]
		v, _, _ := Expand(c[f.Key], tokens, f.IsURLKind())
		values[f.Key] = strings.TrimSpace(v)
	}
	values["assets.base"] = assets

	// megszólítás: ha a benne szereplő változó üres (pl. nincs név), vagy a partner a tartalék
	// megszólítást kapja (B2B cégnév), a tartalék szöveg
	if g, _, empty := Expand(c["note.greeting"], tokens, false); p.FallbackGreeting || len(empty) > 0 || strings.TrimSpace(g) == "" {
		fb, _, _ := Expand(c["note.greetingFallback"], tokens, false)
		values["note.greeting"] = strings.TrimSpace(fb)
	}

	// preheader kiegészítés a kérdéses sablonnál
	if tpl != nil && tpl.HasPoll && values["poll.question"] != "" && values["meta.preheaderPoll"] != "" {
		values["meta.preheader"] = strings.TrimSpace(values["meta.preheader"] + " " + values["meta.preheaderPoll"])
	}

	// képviselő
	values["rep.name"] = strings.TrimSpace(p.RepName)
	values["rep.initials"] = Initials(p.RepName)
	values["rep.photo"] = ""
	if ValidPhoto(p.RepPhoto) {
		photo, _, _ := Expand(strings.TrimSpace(p.RepPhoto), tokens, true)
		values["rep.photo"] = photo
	}
	values["rep.region"] = strings.TrimSpace(p.RepRegion)
	values["rep.phone"], values["rep.phone.url"] = "", ""
	if strings.TrimSpace(p.RepPhone) != "" {
		disp := PhoneDisplay(p.RepPhone)
		if pre := strings.TrimSpace(c["rep.phonePrefix"]); pre != "" {
			disp = pre + " " + disp
		}
		values["rep.phone"] = disp
		values["rep.phone.url"] = PhoneURL(p.RepPhone)
	}
	values["rep.email"], values["rep.email.url"] = "", ""
	if e := strings.TrimSpace(p.RepEmail); e != "" && ValidEmail(e) {
		values["rep.email"] = e
		values["rep.email.url"] = "mailto:" + e
	}

	// link-kiegészítés (az automatikus bejelentkezés a legvégére kerül)
	params, _, _ := Expand(c["offer.linkParams"], tokens, false)
	login := ""
	if strings.TrimSpace(c[AutoLoginKey]) == "1" {
		login = LoginParam(p)
	}
	values["offer.more.url"] = AddLogin(AppendParams(values["offer.more.url"], params), login)

	// partnerenkénti felülírások az Excelből (oszlopfejléc = sablonkulcs)
	for k, v := range p.Overrides {
		if strings.TrimSpace(v) == "" {
			continue
		}
		f := FieldByKey(k)
		urlMode := strings.HasSuffix(k, ".url") || strings.HasSuffix(k, ".image") || k == "rep.photo" || (f != nil && f.IsURLKind())
		ev, _, _ := Expand(v, tokens, urlMode)
		values[k] = ev
	}

	// a partner saját leiratkozó linkje (B2B partnertörzs): változatlanul, paraméter nélkül
	if u := strings.TrimSpace(p.Unsubscribe); u != "" {
		values["footer.unsubscribe.url"] = u
	}

	items := make([]map[string]string, 0, len(sel))
	for _, pr := range sel {
		it := BuildItem(c, pr, tokens, params)
		it["url"] = AddLogin(it["url"], login)
		items = append(items, it)
	}
	return &RenderData{Values: values, Items: items}
}

// AutoLoginKey a közös tartalom kapcsolója ("1" = be): a saját domainre mutató termék- és
// „teljes ajánlat” linkek végére a partner tokenje kerül (&v=<Token>), ami a B2B partnert
// automatikusan bejelentkezteti. Token nélküli partnernél a paraméter kimarad.
const AutoLoginKey = "offer.autoLogin"

var (
	loginTokenRx = regexp.MustCompile(`^[0-9A-Za-z_-]{1,64}$`)
	ownDomainRx  = regexp.MustCompile(`(^|\.)energofish\.[a-z]{2,}$`)
	loginTailRx  = regexp.MustCompile(`[?&]v=[0-9A-Za-z_-]{1,64}$`)
)

// LoginParam a partner olvasói paramétere („v=<Token>”); üres, ha nincs (érvényes) tokenje.
// A token titkos: a programban csak a levél linkjeibe kerül, naplóba és hibaüzenetbe nem.
func LoginParam(p *Partner) string {
	if p == nil || !loginTokenRx.MatchString(p.Token) {
		return ""
	}
	return "v=" + p.Token
}

// OwnDomain igaz, ha a link az Energofish saját domainjére (vagy aldomainjére) mutat.
func OwnDomain(u string) bool {
	pu, err := url.Parse(strings.TrimSpace(u))
	if err != nil || (pu.Scheme != "https" && pu.Scheme != "http") {
		return false
	}
	return ownDomainRx.MatchString(strings.ToLower(pu.Hostname()))
}

// AddLogin a link legvégére teszi az olvasói paramétert (csak saját domainnél).
func AddLogin(u, login string) string {
	if login == "" || !OwnDomain(u) {
		return u
	}
	return AppendParams(u, login)
}

// AppendKeepLogin a paramétereket a link végén álló olvasói paraméter (v=…) elé fűzi, hogy az
// maradjon a legvégén (a Postmark-küldés kampány-UTM-je így is a token elé kerül).
func AppendKeepLogin(u, params string) string {
	frag := ""
	if i := strings.IndexByte(u, '#'); i >= 0 {
		u, frag = u[:i], u[i:]
	}
	loc := loginTailRx.FindStringIndex(u)
	if loc == nil || strings.TrimSpace(params) == "" {
		return AppendParams(u+frag, params)
	}
	return AppendParams(AppendParams(u[:loc[0]], params), u[loc[0]+1:]) + frag
}

// BuildItem egy terméktégla értékei.
func BuildItem(c Content, pr Product, tokens map[string]string, params string) map[string]string {
	t := make(map[string]string, len(tokens)+1)
	for k, v := range tokens {
		t[k] = v
	}
	t["cikkszam"] = strings.TrimSpace(pr.Code)
	link := strings.TrimSpace(pr.URL)
	if link == "" {
		link = c["offer.urlPattern"]
	}
	img := strings.TrimSpace(pr.Image)
	if img == "" {
		img = c["offer.imagePattern"]
	}
	link, _, _ = Expand(link, t, true)
	img, _, _ = Expand(img, t, true)
	alt := strings.TrimSpace(pr.Alt)
	if alt == "" {
		alt = strings.TrimSpace(pr.Name)
	}
	cta := strings.TrimSpace(pr.CTA)
	if cta == "" {
		cta = strings.TrimSpace(c["offer.cta"])
	}
	return map[string]string{
		"url":      AppendParams(link, params),
		"image":    img,
		"imageAlt": alt,
		"code":     strings.TrimSpace(pr.Code),
		"name":     strings.TrimSpace(pr.Name),
		"desc":     strings.TrimSpace(pr.Desc),
		"price":    strings.TrimSpace(pr.Price),
		"deal":     strings.TrimSpace(pr.Deal),
		"cta":      cta,
	}
}
