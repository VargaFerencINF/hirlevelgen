package hirlevel

import (
	"strconv"
	"strings"
)

// SelectedProducts a bekapcsolt termékek.
func SelectedProducts(products []Product) []Product {
	out := make([]Product, 0, len(products))
	for _, p := range products {
		if p.On {
			out = append(out, p)
		}
	}
	return out
}

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

	// megszólítás: ha a benne szereplő változó üres (pl. nincs név), a tartalék szöveg
	if g, _, empty := Expand(c["note.greeting"], tokens, false); len(empty) > 0 || strings.TrimSpace(g) == "" {
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

	// link-kiegészítés
	params, _, _ := Expand(c["offer.linkParams"], tokens, false)
	values["offer.more.url"] = AppendParams(values["offer.more.url"], params)

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

	items := make([]map[string]string, 0, len(sel))
	for _, pr := range sel {
		items = append(items, BuildItem(c, pr, tokens, params))
	}
	return &RenderData{Values: values, Items: items}
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
