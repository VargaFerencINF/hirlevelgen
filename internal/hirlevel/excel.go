package hirlevel

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Column egy felismert (vagy fel nem ismert) Excel-oszlop.
type Column struct {
	Index  int    `json:"index"`
	Letter string `json:"letter"`
	Header string `json:"header"`
	Field  string `json:"field"`            // belső mezőnév; "override" = sablonkulcs; "extra" = változó
	Label  string `json:"label"`            // magyar leírás a felületre
	Target string `json:"target,omitempty"` // override kulcs vagy változónév
}

// ExcelData a beolvasott munkafüzet.
type ExcelData struct {
	Path           string    `json:"path"`
	FileName       string    `json:"fileName"`
	ModTime        time.Time `json:"modTime"`
	LoadedAt       time.Time `json:"loadedAt"`
	Sheets         []string  `json:"sheets"`
	PartnerSheet   string    `json:"partnerSheet"`
	ProductSheet   string    `json:"productSheet"`
	PartnerColumns []Column  `json:"partnerColumns"`
	ProductColumns []Column  `json:"productColumns"`
	Partners       []Partner `json:"partners"`
	Products       []Product `json:"products"`
	Issues         []Issue   `json:"issues"`
	// Source: "" = Excel-fájl, "b2b" = a B2B partnertörzsből összeállított halmaz
	Source string      `json:"source,omitempty"`
	B2B    *B2BSetInfo `json:"b2b,omitempty"`
}

var partnerFieldLabels = map[string]string{
	"email": "Partner e-mail", "name": "Partner neve (megszólítás)", "company": "Cégnév",
	"repName": "Területi képviselő neve", "repPhoto": "Képviselő kép link", "repPhone": "Képviselő telefon",
	"repEmail": "Képviselő e-mail", "repRegion": "Képviselő területe",
}

var productFieldLabels = map[string]string{
	"code": "Cikkszám", "name": "Cikknév", "image": "Cikk kép link", "url": "Gomb link", "desc": "Rövid leírás",
	"price": "Ár", "deal": "Akció", "alt": "Kép helyettesítő szöveg", "cta": "Gombfelirat", "on": "Szerepel a hírlevélben",
}

var overrideKeyRe = regexp.MustCompile(`^[a-z]+(\.[A-Za-z0-9]+)+$`)

// partnerField egy partner-oszlop fejlécét értelmezi.
func partnerField(header string) (field, target string) {
	raw := strings.TrimSpace(header)
	if raw == "" {
		return "", ""
	}
	if overrideKeyRe.MatchString(raw) {
		return "override", raw
	}
	n := Norm(raw)
	if n == "" {
		return "", ""
	}
	for _, rep := range []string{"teruletikepviselo", "kepviselo", "uzletkoto", "salesrep"} {
		if i := strings.Index(n, rep); i >= 0 {
			rest := n[:i] + n[i+len(rep):]
			switch {
			case containsAny(rest, "kep", "foto", "photo", "avatar", "portre", "link", "url"):
				return "repPhoto", ""
			case containsAny(rest, "telefon", "mobil", "phone") || strings.HasPrefix(rest, "tel"):
				return "repPhone", ""
			case containsAny(rest, "email", "mail"):
				return "repEmail", ""
			case containsAny(rest, "terulet", "regio", "megye", "korzet"):
				return "repRegion", ""
			default:
				return "repName", ""
			}
		}
	}
	switch {
	case containsAny(n, "email", "mail"):
		return "email", ""
	case containsAny(n, "cegnev", "company", "cegneve") || n == "ceg" || n == "partnerceg":
		return "company", ""
	case containsAny(n, "nev", "megszolitas", "kapcsolattarto", "name"):
		return "name", ""
	}
	return "extra", n
}

// productField egy termék-oszlop fejlécét értelmezi.
func productField(header string) string {
	n := Norm(header)
	switch {
	case n == "":
		return ""
	case containsAny(n, "aktiv", "szerepel", "hirlevelben", "bekapcsol", "kivalaszt") || n == "kell":
		return "on"
	case containsAny(n, "cikkszam", "termekkod", "cikkkod", "sku", "azonosito") || n == "kod":
		return "code"
	case strings.Contains(n, "alt") && containsAny(n, "kep", "szoveg"):
		return "alt"
	case containsAny(n, "kep", "foto", "image", "img"):
		return "image"
	case containsAny(n, "felirat", "cta", "gombszoveg"):
		return "cta"
	case containsAny(n, "gomb", "link", "url", "termekoldal", "webshop"):
		return "url"
	case containsAny(n, "leiras", "desc"):
		return "desc"
	case containsAny(n, "akcio", "kedvezmeny", "deal", "kiemeles"):
		return "deal"
	case n == "ar" || (strings.HasPrefix(n, "ar") && len(n) <= 6 && !strings.HasPrefix(n, "aru")) ||
		containsAny(n, "ajanlottar", "fogyasztoiar", "nettoar", "bruttoar", "egysegar", "listaar", "price"):
		return "price"
	case containsAny(n, "nev", "megnevezes", "name"):
		return "name"
	}
	return ""
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}

func isIgnoredSheet(name string) bool {
	return containsAny(Norm(name), "utmutato", "segitseg", "leiras", "info", "help", "readme", "jelmagyarazat", "magyarazat")
}

// ReadExcel beolvassa a munkafüzetet (xlsx/xlsm).
func ReadExcel(data []byte, path string, modTime time.Time) (*ExcelData, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".xls" {
		return nil, errors.New("a régi .xls formátum nem támogatott: nyisd meg Excelben, és mentsd el .xlsx formátumban")
	}
	if ext == ".csv" {
		return nil, errors.New("CSV helyett .xlsx munkafüzetet adj meg (Partnerek és Termékek munkalappal)")
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("a fájl nem olvasható Excel-munkafüzetként: %w", err)
	}
	defer f.Close()

	out := &ExcelData{Path: path, FileName: filepath.Base(path), ModTime: modTime, LoadedAt: time.Now()}
	out.Sheets = f.GetSheetList()

	type sheetInfo struct {
		name      string
		rows      [][]string
		pHeader   int
		prHeader  int
		byName    string
		ignorable bool
	}
	var infos []*sheetInfo
	for _, name := range out.Sheets {
		rows, err := f.GetRows(name, excelize.Options{RawCellValue: true})
		if err != nil {
			continue
		}
		si := &sheetInfo{name: name, rows: rows, pHeader: -1, prHeader: -1, ignorable: isIgnoredSheet(name)}
		n := Norm(name)
		switch {
		case containsAny(n, "partner", "cimzett", "cimlista", "vevo", "ugyfel"):
			si.byName = "partner"
		case containsAny(n, "termek", "cikk", "ajanlat", "product"):
			si.byName = "product"
		}
		si.pHeader = findHeader(rows, true)
		si.prHeader = findHeader(rows, false)
		infos = append(infos, si)
	}

	var ps, prs *sheetInfo
	for _, si := range infos { // név szerint
		if ps == nil && si.byName == "partner" && si.pHeader >= 0 {
			ps = si
		}
	}
	for _, si := range infos { // tartalom szerint
		if ps == nil && !si.ignorable && si.pHeader >= 0 {
			ps = si
		}
	}
	for _, si := range infos {
		if prs == nil && si != ps && si.byName == "product" && si.prHeader >= 0 {
			prs = si
		}
	}
	for _, si := range infos {
		if prs == nil && si != ps && !si.ignorable && si.prHeader >= 0 {
			prs = si
		}
	}

	if ps == nil {
		out.Issues = append(out.Issues, Issue{Level: LevelError, Scope: ScopeExcel, Index: -1,
			Message: "Nem található partnerlista: kell egy munkalap, amelynek fejlécében van e-mail oszlop (pl. „Partner e-mail”)."})
	} else {
		out.PartnerSheet = ps.name
		readPartners(f, ps.name, ps.rows, ps.pHeader, out)
	}
	if prs == nil {
		out.Issues = append(out.Issues, Issue{Level: LevelWarn, Scope: ScopeExcel, Index: -1,
			Message: "Nem található terméklista (Termékek munkalap Cikkszám és Cikknév oszloppal). A termékek az appban is megadhatók."})
	} else {
		out.ProductSheet = prs.name
		readProducts(f, prs.name, prs.rows, prs.prHeader, out)
	}
	return out, nil
}

// findHeader az első 20 sorban keresi a fejlécet.
func findHeader(rows [][]string, partner bool) int {
	for i := 0; i < len(rows) && i < 20; i++ {
		fields := map[string]bool{}
		for _, cell := range rows[i] {
			if partner {
				fl, _ := partnerField(cell)
				fields[fl] = true
			} else {
				fields[productField(cell)] = true
			}
		}
		if partner && fields["email"] && (fields["repName"] || fields["name"] || fields["company"] || len(fields) >= 3) {
			return i
		}
		if !partner && fields["code"] && (fields["name"] || fields["image"] || fields["url"]) {
			return i
		}
	}
	return -1
}

func colLetter(i int) string {
	s, _ := excelize.ColumnNumberToName(i + 1)
	return s
}

var hyperlinkFormula = regexp.MustCompile(`(?i)HYPERLINK\(\s*"([^"]+)"`)

// cellLink ha a cella szövege nem link, a cellához rendelt hivatkozást vagy HYPERLINK-képletet adja.
func cellLink(f *excelize.File, sheet string, row, col int, value string) string {
	v := strings.TrimSpace(value)
	if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "{") {
		return v
	}
	cell, err := excelize.CoordinatesToCellName(col+1, row+1)
	if err != nil {
		return v
	}
	if ok, target, err := f.GetCellHyperLink(sheet, cell); err == nil && ok && target != "" {
		return strings.TrimSpace(target)
	}
	if formula, err := f.GetCellFormula(sheet, cell); err == nil && formula != "" {
		if m := hyperlinkFormula.FindStringSubmatch(formula); m != nil {
			return m[1]
		}
	}
	return v
}

func rowEmpty(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func cleanCell(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = strings.ReplaceAll(s, "_x000D_", "")
	return strings.TrimSpace(s)
}

// numericText a nyers számértéket (pl. 3.6301234567E10) egész alakra hozza.
func numericText(s string) string {
	if strings.ContainsAny(s, "Ee") {
		if f, err := strconv.ParseFloat(s, 64); err == nil && f == float64(int64(f)) {
			return strconv.FormatInt(int64(f), 10)
		}
	}
	return s
}

func readPartners(f *excelize.File, sheet string, rows [][]string, header int, out *ExcelData) {
	cols := make([]Column, 0, len(rows[header]))
	seen := map[string]bool{}
	for i, h := range rows[header] {
		field, target := partnerField(h)
		c := Column{Index: i, Letter: colLetter(i), Header: strings.TrimSpace(h), Field: field, Target: target}
		switch field {
		case "":
			continue
		case "override":
			c.Label = "Sablonmező felülírása: " + target
		case "extra":
			c.Label = "Változó: {" + target + "}"
		default:
			if seen[field] {
				c.Field, c.Target, c.Label = "extra", Norm(h), "Ismétlődő oszlop, változóként: {"+Norm(h)+"}"
			} else {
				c.Label = partnerFieldLabels[field]
			}
			seen[field] = true
		}
		cols = append(cols, c)
	}
	out.PartnerColumns = cols
	if !seen["repName"] {
		out.Issues = append(out.Issues, Issue{Level: LevelWarn, Scope: ScopeExcel, Index: -1,
			Message: "A partnerlistán nincs „Területi képviselő” oszlop, így a képviselő-blokk üres marad."})
	}
	for r := header + 1; r < len(rows); r++ {
		row := rows[r]
		if rowEmpty(row) {
			continue
		}
		p := Partner{Row: r + 1}
		for _, c := range cols {
			if c.Index >= len(row) {
				continue
			}
			v := cleanCell(row[c.Index])
			switch c.Field {
			case "email":
				p.Email = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(v, "mailto:"), "MAILTO:"))
			case "name":
				p.Name = v
			case "company":
				p.Company = v
			case "repName":
				p.RepName = v
			case "repPhoto":
				p.RepPhoto = cellLink(f, sheet, r, c.Index, v)
			case "repPhone":
				p.RepPhone = numericText(v)
			case "repEmail":
				p.RepEmail = strings.TrimPrefix(v, "mailto:")
			case "repRegion":
				p.RepRegion = v
			case "override":
				if v != "" {
					if p.Overrides == nil {
						p.Overrides = map[string]string{}
					}
					p.Overrides[c.Target] = v
				}
			case "extra":
				if p.Extra == nil {
					p.Extra = map[string]string{}
				}
				p.Extra[c.Target] = v
			}
		}
		out.Partners = append(out.Partners, p)
	}
	if len(out.Partners) == 0 {
		out.Issues = append(out.Issues, Issue{Level: LevelError, Scope: ScopeExcel, Index: -1,
			Message: fmt.Sprintf("A(z) „%s” munkalapon a fejléc alatt nincs egyetlen partner sem.", sheet)})
	}
}

var falsy = map[string]bool{"nem": true, "n": true, "no": true, "0": true, "false": true, "hamis": true}

func readProducts(f *excelize.File, sheet string, rows [][]string, header int, out *ExcelData) {
	cols := make([]Column, 0, len(rows[header]))
	seen := map[string]bool{}
	for i, h := range rows[header] {
		field := productField(h)
		if field == "" || seen[field] {
			if strings.TrimSpace(h) != "" {
				cols = append(cols, Column{Index: i, Letter: colLetter(i), Header: strings.TrimSpace(h), Label: "Nem használt oszlop"})
			}
			continue
		}
		seen[field] = true
		cols = append(cols, Column{Index: i, Letter: colLetter(i), Header: strings.TrimSpace(h), Field: field, Label: productFieldLabels[field]})
	}
	out.ProductColumns = cols
	for r := header + 1; r < len(rows); r++ {
		row := rows[r]
		if rowEmpty(row) {
			continue
		}
		p := Product{On: true, Row: r + 1}
		for _, c := range cols {
			if c.Field == "" || c.Index >= len(row) {
				continue
			}
			v := cleanCell(row[c.Index])
			switch c.Field {
			case "code":
				p.Code = numericText(v)
			case "name":
				p.Name = v
			case "image":
				p.Image = cellLink(f, sheet, r, c.Index, v)
			case "url":
				p.URL = cellLink(f, sheet, r, c.Index, v)
			case "desc":
				p.Desc = v
			case "price":
				p.Price = FormatPrice(v)
			case "deal":
				p.Deal = v
			case "alt":
				p.Alt = v
			case "cta":
				p.CTA = v
			case "on":
				p.On = !falsy[Norm(v)]
			}
		}
		if p.Code == "" && p.Name == "" {
			continue
		}
		out.Products = append(out.Products, p)
	}
}
