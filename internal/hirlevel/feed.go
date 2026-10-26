package hirlevel

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// Az Energofish nagyker termékfeed (termekadatok_hu.xml) feldolgozása.
// Szerkezet: docs/energofish_termekfeed_struktura_spec.xlsx – gyökér <xml>, rekord <Termek>
// (egy cikk = egy variáns), 37 gyerekelem fix sorrendben, CDATA-s szövegek, a képek
// <Kepek><Kep id="thumb|code|small|large|galleryN">URL</Kep>…</Kepek> alakban (üres is lehet).

// DefaultFeedURL a cikktörzs alapértelmezett címe.
const DefaultFeedURL = "https://energofish.hu/listak/feeds/wholesale/termekadatok_hu.xml"

// FeedImage a cikk egy képe.
type FeedImage struct {
	ID  string `json:"id"`  // thumb, code, small, large, gallery1…
	URL string `json:"url"` // teljes https cím
}

// FeedParam egy kitöltött paraméter (pl. Méret: 12 cm).
type FeedParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// FeedProduct a hírlevélhez szükséges cikkadatok.
type FeedProduct struct {
	Code         string      `json:"code"`  // CikkszamK (kötőjeles, pl. 10000-327)
	Plain        string      `json:"plain"` // Cikkszam (kötőjel nélkül)
	Name         string      `json:"name"`  // Termeknev (variáns)
	Family       string      `json:"family"`
	FamilyID     int         `json:"familyId"`
	Link         string      `json:"link"` // a főtermék webshop oldala
	Category     string      `json:"category"`
	Subcategory  string      `json:"subcategory"`
	Brand        string      `json:"brand"`
	Wholesale    float64     `json:"wholesale"`         // nagyker nettó (a cikktörzs pénznemében)
	WholesaleAkc float64     `json:"wholesaleSale"`     // nagyker nettó akciós (0 = nincs)
	Retail       float64     `json:"retail"`            // kisker bruttó
	RetailAkc    float64     `json:"retailSale"`        // kisker bruttó akciós (0 = nincs)
	Country      string      `json:"country,omitempty"` // a cikktörzs országa (hu, cz, …)
	Stock        int         `json:"stock"`             // 0 = nincs, 1 = van, 2 = ismeretlen jelentés
	LowStock     bool        `json:"lowStock"`
	Badge        string      `json:"badge,omitempty"` // kifutó / készletcsökkentett / díjas
	Images       []FeedImage `json:"images"`
	Params       []FeedParam `json:"params,omitempty"`
	search       string      // normalizált keresőszöveg (minden mező)
	nameKey      string      // normalizált cikkszám + név (az erősebb találathoz)
	codeKey      string      // normalizált cikkszám (kötőjel nélkül)
}

type xmlText struct {
	ID    string `xml:"id,attr"`
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

type xmlTermek struct {
	Cikkszam      string    `xml:"Cikkszam"`
	CikkszamK     string    `xml:"CikkszamK"`
	Termeknev     string    `xml:"Termeknev"`
	Fotermek      xmlText   `xml:"Fotermek"`
	TermekLink    string    `xml:"Termek_link"`
	Kategoria     xmlText   `xml:"Kategoria"`
	Alkategoria   xmlText   `xml:"Alkategoria"`
	Marka         string    `xml:"Marka_gyarto"`
	NagykerNetto  string    `xml:"Nagyker_netto"`
	NagykerAkc    string    `xml:"Nagyker_akcios_netto"`
	KiskerBrutto  string    `xml:"Kisker_brutto"`
	KiskerAkc     string    `xml:"Kisker_akcios_brutto"`
	Keszleten     string    `xml:"Keszleten"`
	AlacsonyKeszl string    `xml:"Alacsony_keszlet"`
	Kepek         []xmlText `xml:"Kepek>Kep"`
	Parameterek   []xmlText `xml:"Parameterek>Parameter"`
	FotermekMatr  string    `xml:"Fotermek_matrica"`
	AltermekMatr  string    `xml:"Altermek_matrica"`
}

// xmlProduct a külföldi cikktörzsek (AT, CZ, SK, EN, RO, ES) rekordja: ugyanazok az adatok angol
// mezőnevekkel (<Product>, Item_codeK, Product_name, Main_product, Wholesale_net_price, …).
type xmlProduct struct {
	ItemCode      string    `xml:"Item_code"`
	ItemCodeK     string    `xml:"Item_codeK"`
	ProductName   string    `xml:"Product_name"`
	MainProduct   xmlText   `xml:"Main_product"`
	ProductLink   string    `xml:"Product_link"`
	Category      xmlText   `xml:"Category"`
	Subcategory   xmlText   `xml:"Subcategory"`
	Brand         string    `xml:"Brand_manufacturer"`
	WholesaleNet  string    `xml:"Wholesale_net_price"`
	WholesaleDisc string    `xml:"Wholesale_discount_net_price"`
	RetailGross   string    `xml:"Retail_gross_price"`
	RetailDisc    string    `xml:"Retail_discount_gross_price"`
	OnStock       string    `xml:"On_stock"`
	LowStock      string    `xml:"Low_stock"`
	Images        []xmlText `xml:"Images>Image"`
	Parameters    []xmlText `xml:"Parameters>Parameter"`
	ProductStick  string    `xml:"Product_sticker"`
	SubStick      string    `xml:"Subproduct_sticker"`
}

func (x *xmlProduct) termek() *xmlTermek {
	return &xmlTermek{Cikkszam: x.ItemCode, CikkszamK: x.ItemCodeK, Termeknev: x.ProductName, Fotermek: x.MainProduct,
		TermekLink: x.ProductLink, Kategoria: x.Category, Alkategoria: x.Subcategory, Marka: x.Brand,
		NagykerNetto: x.WholesaleNet, NagykerAkc: x.WholesaleDisc, KiskerBrutto: x.RetailGross, KiskerAkc: x.RetailDisc,
		Keszleten: x.OnStock, AlacsonyKeszl: x.LowStock, Kepek: x.Images, Parameterek: x.Parameters,
		FotermekMatr: x.ProductStick, AltermekMatr: x.SubStick}
}

// a rövid leírásba nem kerülő paraméterek azonosítója (minden nyelvű cikktörzsben azonos):
// 5 cikkszám, 8 egyéb, 29 szállítási méret, 35 tartalom, 40 akció
var paramSkipID = map[string]bool{"5": true, "8": true, "29": true, "35": true, "40": true}

// fallbackName a név nélküli cikk neve (a külföldi cikktörzsekben a cikkek egy részének üres a
// Product_name mezője): a főtermék neve és a változat mérete (21), ha az üres, a kiszerelése (19),
// a színe (31) vagy más jellemzője – pl. „MUSTAD KAIJU SINGLE RING TITAN STEEL HÁČEK 8/0”.
func fallbackName(t *xmlTermek) string {
	base := strings.Join(strings.Fields(t.Fotermek.Value), " ")
	variant := ""
	for _, id := range []string{"21", "19", "31", ""} { // "" = bármely más (a leírásba illő) paraméter
		for _, prm := range t.Parameterek {
			pid := clean(prm.ID)
			if (pid == id || (id == "" && !paramSkipID[pid])) && clean(prm.Value) != "" {
				variant = strings.Join(strings.Fields(prm.Value), " ")
				break
			}
		}
		if variant != "" {
			break
		}
	}
	switch {
	case base == "":
		return variant
	case variant == "" || strings.Contains(strings.ToLower(base), strings.ToLower(variant)):
		return base
	}
	return base + " " + variant
}

// clean a spec szerinti strip (a nem törő szóközt is beleértve).
func clean(s string) string { return strings.TrimFunc(s, unicode.IsSpace) }

func atoi(s string) int {
	n, _ := strconv.Atoi(clean(s))
	return n
}

// cleanURL a szóközt tartalmazó kép-URL-eket kódolja (V35).
func cleanURL(s string) string { return strings.ReplaceAll(clean(s), " ", "%20") }

func badgeOf(files ...string) string {
	for _, f := range files {
		f = strings.ToLower(f)
		switch {
		case strings.Contains(f, "kifuto"):
			return "kifutó"
		case strings.Contains(f, "keszletcsokkentett"):
			return "készletcsökkentett"
		case strings.Contains(f, "efttex"):
			return "EFTTEX díjas"
		}
	}
	return ""
}

func convertTermek(t *xmlTermek) FeedProduct {
	p := FeedProduct{
		Code: clean(t.CikkszamK), Plain: clean(t.Cikkszam), Name: strings.Join(strings.Fields(t.Termeknev), " "),
		Family: clean(t.Fotermek.Value), FamilyID: atoi(t.Fotermek.ID), Link: clean(t.TermekLink),
		Category: clean(t.Kategoria.Value), Subcategory: clean(t.Alkategoria.Value), Brand: clean(t.Marka),
		Wholesale: parseNum(t.NagykerNetto), WholesaleAkc: parseNum(t.NagykerAkc),
		Retail: parseNum(t.KiskerBrutto), RetailAkc: parseNum(t.KiskerAkc),
		Stock: atoi(t.Keszleten), LowStock: clean(t.AlacsonyKeszl) == "1",
		Badge: badgeOf(clean(t.AltermekMatr), clean(t.FotermekMatr)),
	}
	if p.Code == "" {
		p.Code = p.Plain
	}
	if p.Name == "" {
		p.Name = fallbackName(t)
	}
	for _, k := range t.Kepek {
		if u := cleanURL(k.Value); u != "" {
			p.Images = append(p.Images, FeedImage{ID: clean(k.ID), URL: u})
		}
	}
	for _, prm := range t.Parameterek {
		v := clean(prm.Value)
		if v == "" || paramSkipID[clean(prm.ID)] { // cikkszám, akció, szállítási méret … (a leírásba nem kell)
			continue
		}
		p.Params = append(p.Params, FeedParam{Name: clean(prm.Name), Value: v})
	}
	p.codeKey = strings.ReplaceAll(Norm(p.Code), "-", "")
	p.nameKey = searchNorm(p.Code + " " + p.Plain + " " + p.Name)
	p.search = searchNorm(strings.Join([]string{p.Code, p.Plain, p.Name, p.Family, p.Brand, p.Category, p.Subcategory}, " "))
	return p
}

// searchNorm kisbetűs, ékezet nélküli, a betű-szám sorozatokat szóközzel elválasztó alak.
func searchNorm(s string) string {
	s = strings.ToLower(accentMap.Replace(s))
	var b strings.Builder
	space := true
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			space = false
		} else if !space {
			b.WriteByte(' ')
			space = true
		}
	}
	return " " + strings.TrimSpace(b.String()) + " "
}

// charsetReader a nem UTF-8 feedekhez (a régi magyar kódlapok).
func charsetReader(label string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(strings.ReplaceAll(label, "_", "-")) {
	case "iso-8859-2", "latin2", "latin-2", "iso8859-2":
		return charmap.ISO8859_2.NewDecoder().Reader(input), nil
	case "windows-1250", "cp1250", "win-1250":
		return charmap.Windows1250.NewDecoder().Reader(input), nil
	case "iso-8859-1", "latin1", "latin-1", "windows-1252", "cp1252":
		return charmap.Windows1252.NewDecoder().Reader(input), nil
	}
	return nil, fmt.Errorf("a cikktörzs karakterkódolása (%s) nem támogatott", label)
}

// ParseFeed a teljes feedet folyamatosan (streaming) olvassa be.
func ParseFeed(r io.Reader) ([]FeedProduct, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = true
	dec.CharsetReader = charsetReader
	var out []FeedProduct
	seen := map[string]bool{}
	root := false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("a cikktörzs nem értelmezhető (%d. cikk után): %w", len(out), err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if !root {
			root = true
			if se.Name.Local != "xml" {
				return nil, fmt.Errorf("ismeretlen cikktörzs-formátum: a gyökérelem <%s>, <xml> várt", se.Name.Local)
			}
			continue
		}
		var t *xmlTermek
		switch se.Name.Local {
		case "Termek": // magyar cikktörzs
			t = &xmlTermek{}
			if err := dec.DecodeElement(t, &se); err != nil {
				return nil, fmt.Errorf("hibás cikk a(z) %d. helyen: %w", len(out)+1, err)
			}
		case "Product": // külföldi cikktörzs (angol mezőnevek)
			var x xmlProduct
			if err := dec.DecodeElement(&x, &se); err != nil {
				return nil, fmt.Errorf("hibás cikk a(z) %d. helyen: %w", len(out)+1, err)
			}
			t = x.termek()
		default:
			if err := dec.Skip(); err != nil {
				return nil, err
			}
			continue
		}
		p := convertTermek(t)
		if p.Code == "" || seen[p.Code] {
			continue
		}
		seen[p.Code] = true
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, errors.New("a cikktörzsben nincs egyetlen cikk sem")
	}
	return out, nil
}

// SearchFeed cikkszámra vagy névre keres (ékezet- és kisbetű-független, minden szónak illeszkednie kell).
func SearchFeed(list []FeedProduct, query string, limit int) []FeedProduct {
	q := strings.TrimSpace(query)
	if q == "" || len(list) == 0 {
		return nil
	}
	qcode := strings.ReplaceAll(Norm(q), "-", "")
	words := strings.Fields(searchNorm(q))
	type hit struct {
		i, score int
	}
	var hits []hit
	for i := range list {
		p := &list[i]
		score := -1
		switch {
		case qcode != "" && p.codeKey == qcode:
			score = 0
		case len(qcode) >= 3 && strings.HasPrefix(p.codeKey, qcode):
			score = 1
		}
		if score < 0 && len(words) > 0 {
			// a cikknévben talált szavak előrébb kerülnek, mint a márka/kategória/főtermék egyezései
			switch {
			case containsAll(p.nameKey, words):
				score = 3
				if strings.Contains(p.nameKey, " "+words[0]) {
					score = 2
				}
			case containsAll(p.search, words):
				score = 5
				if strings.Contains(p.search, " "+words[0]) {
					score = 4
				}
			}
		}
		if score >= 0 {
			hits = append(hits, hit{i, score})
		}
	}
	sort.SliceStable(hits, func(a, b int) bool {
		if hits[a].score != hits[b].score {
			return hits[a].score < hits[b].score
		}
		pa, pb := &list[hits[a].i], &list[hits[b].i]
		if (pa.Stock == 0) != (pb.Stock == 0) {
			return pb.Stock == 0
		}
		return pa.Code < pb.Code
	})
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]FeedProduct, len(hits))
	for i, h := range hits {
		out[i] = list[h.i]
	}
	return out
}

func containsAll(s string, words []string) bool {
	for _, w := range words {
		if !strings.Contains(s, w) {
			return false
		}
	}
	return true
}

// FindFeed cikkszám (kötőjellel vagy anélkül) szerint keres.
func FindFeed(list []FeedProduct, code string) *FeedProduct {
	k := strings.ReplaceAll(Norm(code), "-", "")
	for i := range list {
		if list[i].codeKey == k {
			return &list[i]
		}
	}
	return nil
}

// DealText az akció kedvezménye („−23% akció”), ha van.
func DealText(list, sale float64) string {
	return dealText(list, sale, "akció")
}

// dealText az ország akció-szavával (üres szónál csak a százalék: „−23%”).
func dealText(list, sale float64, word string) string {
	if sale <= 0 || list <= 0 || sale >= list {
		return ""
	}
	pct := int(math.Round((list - sale) / list * 100))
	if pct < 1 {
		if word == "" {
			return ""
		}
		return word
	}
	if word == "" {
		return fmt.Sprintf("−%d%%", pct)
	}
	return fmt.Sprintf("−%d%% %s", pct, word)
}

// FeedOptions a cikktörzsből átvett adatok beállításai (a felületen állíthatók, mentődnek).
type FeedOptions struct {
	URL   string `json:"url,omitempty"`   // üres = DefaultFeedURL
	Price string `json:"price,omitempty"` // retail (alap): kisker bruttó; wholesale: nagyker nettó; none
	Image string `json:"image,omitempty"` // az előnyben részesített kép: code (alap), large, small, thumb
	Link  string `json:"link,omitempty"`  // webshop (alap): a feed termékoldala; pattern: a Haladó beállítások mintája
	// Country a cikktörzs országa ("" = a partnerhalmaz célcsoportja szerint, Excelnél magyar)
	Country string `json:"country,omitempty"`
	// Currency országonként felülírt pénznem-jel (pl. {"ro": "RON"}); üres = az alapértelmezett
	Currency map[string]string `json:"currency,omitempty"`
	// URLs országonként saját cikktörzs-cím (üres = az alapértelmezett termekadatok_<kód>.xml);
	// a magyarnál a korábbi URL mező is érvényes
	URLs map[string]string `json:"urls,omitempty"`
}

// URLFor az ország saját cikktörzs-címe ("" = az alapértelmezett).
func (o FeedOptions) URLFor(code string) string {
	if v := strings.TrimSpace(o.URLs[code]); v != "" {
		return v
	}
	if code == "hu" {
		return strings.TrimSpace(o.URL)
	}
	return ""
}

var imageOrder = map[string][]string{
	"code":  {"code", "large", "small", "thumb"},
	"large": {"large", "code", "small", "thumb"},
	"small": {"small", "code", "large", "thumb"},
	"thumb": {"thumb", "small", "code", "large"},
}

// PickImage a kitöltött képek közül az előnyben részesítettet adja (ha az üres, a következőt).
// A WEBP képet (az Outlook nem jeleníti meg) csak akkor választja, ha nincs más.
func PickImage(images []FeedImage, pref string) string {
	order, ok := imageOrder[pref]
	if !ok {
		order = imageOrder["code"]
	}
	for _, allowWebp := range []bool{false, true} {
		for _, id := range order {
			for _, im := range images {
				if im.ID == id && (allowWebp || !IsWebp(im.URL)) {
					return im.URL
				}
			}
		}
		for _, im := range images { // galériakép
			if allowWebp || !IsWebp(im.URL) {
				return im.URL
			}
		}
	}
	return ""
}

// IsWebp igaz, ha a kép WEBP fájlra mutat.
func IsWebp(u string) bool {
	return strings.HasSuffix(strings.ToLower(strings.SplitN(u, "?", 2)[0]), ".webp")
}

// a rövid leírásba nem kerülő paraméterek
var descSkip = map[string]bool{"Cikkszám": true, "Akció": true, "Szállítási méret": true, "Tartalom": true, "Egyéb": true}

// FeedDesc rövid leírás a paraméterekből (pl. „12 cm · Red · Crab”), legfeljebb max karakter;
// paraméter híján „alkategória · márka”.
func FeedDesc(p *FeedProduct, max int) string {
	var parts []string
	n := 0
	add := func(v string) {
		l := utf8.RuneCountInString(v)
		sep := 0
		if len(parts) > 0 {
			sep = 3
		}
		if v == "" || n+sep+l > max {
			return
		}
		parts = append(parts, v)
		n += sep + l
	}
	for _, prm := range p.Params {
		if descSkip[prm.Name] {
			continue
		}
		v := prm.Value
		if !strings.ContainsFunc(v, unicode.IsLetter) { // csupasz szám: a név is kell („Méret: 8”)
			v = prm.Name + ": " + v
		}
		add(v)
	}
	if len(parts) == 0 {
		add(p.Subcategory)
		add(p.Brand)
	}
	return strings.Join(parts, " · ")
}

// feedDeal az akció mező: kedvezmény és/vagy matrica (pl. „−55%, kifutó”).
func feedDeal(list, sale float64, badge, country string) string {
	c := FindFeedCountry(country)
	d := dealText(list, sale, c.Deal)
	if c.Code != "hu" {
		return d // a matricák (kifutó, kiárusítás) magyar szavak: külföldi cikktörzsnél csak a kedvezmény
	}
	tag := map[string]string{"kifutó": "kifutó", "készletcsökkentett": "kiárusítás", "EFTTEX díjas": "EFTTEX-díjas"}[badge]
	switch {
	case d != "" && tag != "":
		return strings.TrimSuffix(d, " akció") + ", " + tag
	case d != "":
		return d
	case tag == "kifutó":
		return "Kifutó termék"
	case tag != "":
		return strings.ToUpper(tag[:1]) + tag[1:]
	}
	return ""
}

// Ref a cikk és a főtermék adatai (a termékkártya cikk/főtermék váltójához).
func (p *FeedProduct) Ref() *FeedRef {
	return &FeedRef{Code: p.Code, Name: p.Name, Family: p.Family, FamilyID: p.FamilyID, Link: p.Link}
}

// ToProduct a cikktörzs adataiból hírlevél-termék (utána szabadon szerkeszthető).
func (p *FeedProduct) ToProduct(o FeedOptions) Product {
	pr := Product{On: true, Code: p.Code, Name: p.Name, Desc: FeedDesc(p, 30), Alt: p.Name,
		Images: append([]FeedImage(nil), p.Images...), Feed: p.Ref()}
	country := FindFeedCountry(p.Country)
	list, sale, suffix := p.Retail, p.RetailAkc, ""
	if o.Price == "wholesale" {
		list, sale, suffix = p.Wholesale, p.WholesaleAkc, country.NetSuffix
	}
	cur := list
	if sale > 0 && (sale < list || list == 0) {
		cur = sale
	}
	if o.Price != "none" && cur > 0 {
		pr.Price = FormatMoney(cur, country.Code, o.Currency[country.Code]) + suffix
	}
	pr.Deal = feedDeal(list, sale, p.Badge, country.Code)
	pr.Image = PickImage(p.Images, o.Image)
	if o.Link != "pattern" {
		pr.URL = p.Link
	}
	return pr
}
