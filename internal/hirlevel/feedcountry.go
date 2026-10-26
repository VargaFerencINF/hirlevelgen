package hirlevel

import (
	"math"
	"strconv"
	"strings"
)

// Országonkénti cikktörzs: ugyanaz a szerkezet (termekadatok_<kód>.xml), az adott ország
// nyelvén, pénznemében és áraival. Hogy melyik az aktív, a betöltött partnerhalmaz
// célcsoportja dönti el (kézzel felülírható); Excel-partnerlistánál a magyar.

// FeedCountry egy ország cikktörzse.
type FeedCountry struct {
	Code      string `json:"code"`      // hu, at, cz, sk, en, ro, es
	Label     string `json:"label"`     // a felületen
	Currency  string `json:"currency"`  // alapértelmezett pénznem-jel (a cikkválasztóban felülírható)
	NetSuffix string `json:"netSuffix"` // a nettó (nagyker) ár után
	Deal      string `json:"deal"`      // az akció szó (a magyarnál „akció”, máshol nincs)
}

// FeedCountries a támogatott cikktörzsek.
var FeedCountries = []FeedCountry{
	{"hu", "Magyar (HU)", "Ft", " + áfa", "akció"},
	{"at", "Osztrák (AT)", "€", " zzgl. MwSt.", ""},
	{"cz", "Cseh (CZ)", "Kč", " bez DPH", ""},
	{"sk", "Szlovák (SK)", "€", " bez DPH", ""},
	{"en", "Nemzetközi (EN)", "€", " + VAT", ""},
	{"ro", "Román (RO)", "lei", " fără TVA", ""},
	{"es", "Spanyol (ES)", "€", " + IVA", ""},
}

// FeedURLFor az ország cikktörzsének címe.
func FeedURLFor(code string) string {
	if code == "" || code == "hu" {
		return DefaultFeedURL
	}
	return "https://energofish.hu/listak/feeds/wholesale/termekadatok_" + code + ".xml"
}

// FindFeedCountry kód szerint (ismeretlen kódra a magyart adja).
func FindFeedCountry(code string) *FeedCountry {
	for i := range FeedCountries {
		if FeedCountries[i].Code == code {
			return &FeedCountries[i]
		}
	}
	return &FeedCountries[0]
}

// FeedCountryForGroup a B2B célcsoport cikktörzse (a német célcsoport az osztrák, a szerb és a
// nemzetközi az angol/nemzetközi cikktörzset kapja).
func FeedCountryForGroup(group string) string {
	switch strings.ToUpper(strings.TrimSpace(group)) {
	case "B2B_AT", "B2B_DE":
		return "at"
	case "B2B_CZ":
		return "cz"
	case "B2B_SK":
		return "sk"
	case "B2B_RO":
		return "ro"
	case "B2B_ES":
		return "es"
	case "B2B_COM", "B2B_RS":
		return "en"
	}
	return "hu"
}

// parseNum a cikktörzs árai: egész vagy tizedes (ponttal vagy vesszővel), szóközzel tagolva is.
func parseNum(s string) float64 {
	s = strings.NewReplacer(" ", "", " ", "", ",", ".").Replace(clean(s))
	if strings.Count(s, ".") > 1 { // ezres pont: 1.290.50 – csak az utolsó a tizedesjel
		i := strings.LastIndex(s, ".")
		s = strings.ReplaceAll(s[:i], ".", "") + s[i:]
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < 0 || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return f
}

// FormatMoney az ár a cikktörzs országa szerint: a magyar „3 090 Ft” (mint eddig), máshol
// tizedesvesszővel, ha nem kerek („12,90 €”, „1 290 Kč”), a nemzetközinél „€12.90”.
// A currency (ha nem üres) felülírja az ország pénznem-jelét.
func FormatMoney(v float64, country, currency string) string {
	c := FindFeedCountry(country)
	if c.Code == "hu" && currency == "" {
		return FormatPrice(strconv.FormatFloat(v, 'f', -1, 64))
	}
	cur := c.Currency
	if strings.TrimSpace(currency) != "" {
		cur = strings.TrimSpace(currency)
	}
	cents := int64(math.Round(v * 100))
	whole, frac := cents/100, cents%100
	digits := strconv.FormatInt(whole, 10)
	sep := " "
	if c.Code == "en" {
		sep = ","
	}
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteString(sep)
		}
		b.WriteRune(r)
	}
	num := b.String()
	if frac != 0 {
		dec := ","
		if c.Code == "en" {
			dec = "."
		}
		num += dec + strconv.FormatInt(100+frac, 10)[1:]
	}
	if c.Code == "en" && (cur == "€" || cur == "$" || cur == "£") {
		return cur + num
	}
	return num + " " + cur
}
