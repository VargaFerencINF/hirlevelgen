package hirlevel

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GenerateOptions a generálás beállításai.
type GenerateOptions struct {
	OutputDir   string `json:"outputDir"`
	EML         bool   `json:"eml"`
	From        string `json:"from"`
	FilePattern string `json:"filePattern"`
	Only        []int  `json:"only,omitempty"` // partner-indexek; üres = mind
}

// SkippedPartner egy kihagyott partner.
type SkippedPartner struct {
	Row    int    `json:"row"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// GenerateResult a generálás eredménye.
type GenerateResult struct {
	Folder    string           `json:"folder"`
	Generated int              `json:"generated"`
	Skipped   []SkippedPartner `json:"skipped"`
	Index     string           `json:"index"`
	CSV       string           `json:"csv"`
	FirstHTML string           `json:"firstHtml"`
	Warnings  []string         `json:"warnings"`
	Millis    int64            `json:"millis"`
}

// DefaultFilePattern a kimeneti fájlnevek alapmintája.
const DefaultFilePattern = "{sorszam}_{email}"

var badFileChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]+`)

// SafeFileName Windows-on is érvényes fájlnév.
func SafeFileName(s string) string {
	s = badFileChars.ReplaceAllString(s, "_")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.Trim(s, " .")
	if r := []rune(s); len(r) > 120 {
		s = string(r[:120])
	}
	switch strings.ToUpper(strings.SplitN(s, ".", 2)[0]) {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "LPT1", "LPT2":
		s = "_" + s
	}
	if s == "" {
		s = "hirlevel"
	}
	return s
}

// FileBaseName a partner kimeneti fájlneve kiterjesztés nélkül.
func FileBaseName(pattern string, p *Partner, seq, total int) string {
	if strings.TrimSpace(pattern) == "" {
		pattern = DefaultFilePattern
	}
	width := len(strconv.Itoa(total))
	if width < 3 {
		width = 3
	}
	t := PartnerTokens(p)
	t["sorszam"] = fmt.Sprintf("%0*d", width, seq)
	v, _, _ := Expand(pattern, t, false)
	return SafeFileName(v)
}

// Generate legenerálja a hírleveleket a kimeneti mappába.
func Generate(assets fs.FS, tpl *Template, c Content, products []Product, partners []Partner, opt GenerateOptions) (*GenerateResult, error) {
	start := time.Now()
	if tpl == nil {
		return nil, errors.New("nincs kiválasztott sablon")
	}
	if strings.TrimSpace(opt.OutputDir) == "" {
		return nil, errors.New("nincs megadva kimeneti mappa")
	}
	for _, is := range append(ValidateContent(c, products, tpl), ValidateProducts(c, products)...) {
		if is.Level == LevelError {
			return nil, fmt.Errorf("a tartalomban hiba van: %s", is.Message)
		}
	}
	var from *mail.Address
	if opt.EML && strings.TrimSpace(opt.From) != "" {
		a, err := mail.ParseAddress(strings.TrimSpace(opt.From))
		if err != nil {
			return nil, fmt.Errorf("érvénytelen feladó (%s): pl. Energofish <hirlevel@energofish.hu>", opt.From)
		}
		from = a
	}

	// kiválasztott partnerek
	var list []int
	if len(opt.Only) > 0 {
		for _, i := range opt.Only {
			if i >= 0 && i < len(partners) {
				list = append(list, i)
			}
		}
	} else {
		for i := range partners {
			list = append(list, i)
		}
	}
	if len(list) == 0 {
		return nil, errors.New("nincs kiválasztott partner")
	}

	if err := os.MkdirAll(opt.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("a kimeneti mappa nem hozható létre: %w", err)
	}
	base := time.Now().Format("2006-01-02_15-04") + "_" + tpl.ID
	folder := filepath.Join(opt.OutputDir, base)
	for n := 2; ; n++ {
		if _, err := os.Stat(folder); os.IsNotExist(err) {
			break
		}
		folder = filepath.Join(opt.OutputDir, fmt.Sprintf("%s_%d", base, n))
	}
	htmlDir := filepath.Join(folder, "html")
	emlDir := filepath.Join(folder, "eml")
	if err := os.MkdirAll(htmlDir, 0o755); err != nil {
		return nil, fmt.Errorf("a kimeneti mappa nem hozható létre: %w", err)
	}
	if opt.EML {
		if err := os.MkdirAll(emlDir, 0o755); err != nil {
			return nil, err
		}
	}

	res := &GenerateResult{Folder: folder}
	localAssets := strings.TrimSpace(c["assets.base"]) == ""
	if localAssets {
		if err := CopyAssets(assets, filepath.Join(folder, "assets")); err != nil {
			return nil, fmt.Errorf("a képtár másolása nem sikerült: %w", err)
		}
		res.Warnings = append(res.Warnings, "A képtár webcíme üres volt, ezért a képek az assets mappába kerültek: a levelek helyben jól látszanak, de kiküldés előtt a képtárat fel kell tölteni.")
		if opt.EML {
			res.Warnings = append(res.Warnings, "Az EML fájlokban a sablon saját képei (logó, hullámok, {assets} képek) csak feltöltött képtárral jelennek meg.")
		}
	}

	type row struct {
		seq              int
		p                *Partner
		htmlRel, emlRel  string
		subject, repName string
	}
	var rows []row
	used := map[string]bool{}
	total := len(list)
	for seq, idx := range list {
		p := &partners[idx]
		if PartnerBlocked(p) {
			reason := "érvénytelen e-mail cím"
			if p.Email == "" {
				reason = "hiányzó e-mail cím"
			}
			res.Skipped = append(res.Skipped, SkippedPartner{Row: p.Row, Email: p.Email, Name: p.Name, Reason: reason})
			continue
		}
		d := Build(c, products, p, tpl, "../assets")
		html, missing := tpl.Render(d)
		if len(missing) > 0 {
			return nil, fmt.Errorf("a sablon ismeretlen mezőket használ: %s", strings.Join(missing, ", "))
		}
		name := FileBaseName(opt.FilePattern, p, seq+1, total)
		for n := 2; used[strings.ToLower(name)]; n++ {
			name = fmt.Sprintf("%s_%d", FileBaseName(opt.FilePattern, p, seq+1, total), n)
		}
		used[strings.ToLower(name)] = true
		r := row{seq: seq + 1, p: p, htmlRel: "html/" + name + ".html", subject: d.Values["meta.subject"], repName: p.RepName}
		if err := os.WriteFile(filepath.Join(htmlDir, name+".html"), []byte(html), 0o644); err != nil {
			return nil, fmt.Errorf("írási hiba: %w", err)
		}
		if opt.EML {
			eml := BuildEML(from, p, d.Values["meta.subject"], html, BuildText(d, tpl))
			if err := os.WriteFile(filepath.Join(emlDir, name+".eml"), eml, 0o644); err != nil {
				return nil, fmt.Errorf("írási hiba: %w", err)
			}
			r.emlRel = "eml/" + name + ".eml"
		}
		if res.FirstHTML == "" {
			res.FirstHTML = filepath.Join(htmlDir, name+".html")
		}
		rows = append(rows, r)
		res.Generated++
	}

	// küldési lista (Excel-barát: UTF-8 BOM, pontosvessző)
	var buf bytes.Buffer
	buf.WriteString("\ufeff")
	w := csv.NewWriter(&buf)
	w.Comma = ';'
	w.UseCRLF = true
	_ = w.Write([]string{"Sorszám", "E-mail", "Név", "Cégnév", "Területi képviselő", "Tárgy", "HTML fájl", "EML fájl"})
	for _, r := range rows {
		_ = w.Write([]string{strconv.Itoa(r.seq), r.p.Email, r.p.Name, r.p.Company, r.repName, r.subject,
			filepath.FromSlash(r.htmlRel), filepath.FromSlash(r.emlRel)})
	}
	w.Flush()
	res.CSV = filepath.Join(folder, "kuldesi-lista.csv")
	if err := os.WriteFile(res.CSV, buf.Bytes(), 0o644); err != nil {
		return nil, err
	}
	if j, err := ExportJSON(c, products); err == nil {
		_ = os.WriteFile(filepath.Join(folder, "tartalom.json"), j, 0o644)
	}

	// áttekintő oldal
	logo, _ := fs.ReadFile(assets, "energofish-mark-light.png")
	var ib strings.Builder
	ib.WriteString(overviewHead(tpl, res, len(rows), logo))
	for _, r := range rows {
		company := ""
		if r.p.Company != "" {
			company = `<div class="sub">` + EscapeHTML(r.p.Company) + `</div>`
		}
		eml := ""
		if r.emlRel != "" {
			eml = ` <a class="btn ghost" href="` + EscapeHTML(r.emlRel) + `">EML</a>`
		}
		fmt.Fprintf(&ib, `<tr><td class="num">%d</td><td><strong>%s</strong>%s</td><td>%s</td><td>%s</td><td class="act"><a class="btn" href="%s" target="_blank">Megnyitás</a>%s</td></tr>`+"\n",
			r.seq, EscapeHTML(orDash(r.p.Name)), company, EscapeHTML(r.p.Email), EscapeHTML(orDash(r.repName)), EscapeHTML(r.htmlRel), eml)
	}
	ib.WriteString("</tbody></table>")
	if len(res.Skipped) > 0 {
		ib.WriteString(`<h2>Kihagyott partnerek</h2><table><thead><tr><th>Excel-sor</th><th>Név</th><th>E-mail</th><th>Ok</th></tr></thead><tbody>`)
		for _, s := range res.Skipped {
			fmt.Fprintf(&ib, "<tr><td class=\"num\">%d</td><td>%s</td><td>%s</td><td>%s</td></tr>\n", s.Row, EscapeHTML(orDash(s.Name)), EscapeHTML(orDash(s.Email)), EscapeHTML(s.Reason))
		}
		ib.WriteString("</tbody></table>")
	}
	ib.WriteString(`<p class="foot">Energofish Partnerhírlevél-generátor · ` + time.Now().Format("2006. 01. 02. 15:04") + `</p></main></body></html>`)
	res.Index = filepath.Join(folder, "attekinto.html")
	if err := os.WriteFile(res.Index, []byte(ib.String()), 0o644); err != nil {
		return nil, err
	}
	res.Millis = time.Since(start).Milliseconds()
	return res, nil
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "–"
	}
	return s
}

func overviewHead(tpl *Template, res *GenerateResult, n int, logo []byte) string {
	img := ""
	if len(logo) > 0 {
		img = `<img src="data:image/png;base64,` + base64.StdEncoding.EncodeToString(logo) + `" alt="Energofish" width="49" height="40">`
	}
	skipped := ""
	if len(res.Skipped) > 0 {
		skipped = fmt.Sprintf(`<div class="stat warn"><b>%d</b><span>kihagyva</span></div>`, len(res.Skipped))
	}
	return `<!DOCTYPE html><html lang="hu"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Áttekintő · ` + EscapeHTML(tpl.Name) + `</title><style>
*{box-sizing:border-box}body{margin:0;font-family:'Open Sans','Segoe UI',Arial,sans-serif;background:#F0F0F0;color:#1A171E}
header{background:#1A171E;color:#fff;padding:22px 40px;display:flex;align-items:center;gap:20px;border-bottom:6px solid #F1A32B}
header h1{margin:0;font-size:20px}header p{margin:2px 0 0;color:#CCCCCC;font-size:13px}
main{max-width:1100px;margin:0 auto;padding:28px 24px 60px}
.stats{display:flex;gap:12px;margin-bottom:20px;flex-wrap:wrap}.stat{background:#fff;border:1px solid #E6E6E6;border-radius:10px;padding:12px 18px;min-width:130px}
.stat b{display:block;font-size:26px}.stat span{font-size:12px;font-weight:700;letter-spacing:.06em;text-transform:uppercase;color:#555}.stat.warn b{color:#9E5F0C}
h2{font-size:12px;letter-spacing:.06em;text-transform:uppercase;margin:28px 0 10px}
table{width:100%;border-collapse:collapse;background:#fff;border:1px solid #E6E6E6;border-radius:10px;overflow:hidden;font-size:14px}
th{text-align:left;font-size:11px;letter-spacing:.06em;text-transform:uppercase;color:#555;background:#F5F5F5;padding:10px 12px}
td{padding:10px 12px;border-top:1px solid #EFEFEF;vertical-align:top}.num{color:#555;width:56px}.sub{color:#555;font-size:12px}
.act{white-space:nowrap;text-align:right}.btn{display:inline-block;background:#F1A32B;color:#1A171E;font-weight:700;text-decoration:none;padding:5px 12px;border-radius:8px;font-size:13px}
.btn.ghost{background:transparent;border:2px solid #414242;padding:3px 10px}.foot{color:#555;font-size:12px;margin-top:24px}
</style></head><body><header>` + img + `<div><h1>Partnerhírlevél – áttekintő</h1><p>Sablon: ` + EscapeHTML(tpl.Name) + ` (` + tpl.Short + `) · ` + EscapeHTML(filepath.Base(res.Folder)) + `</p></div></header><main>
<div class="stats"><div class="stat"><b>` + strconv.Itoa(n) + `</b><span>hírlevél</span></div>` + skipped + `</div>
<table><thead><tr><th>#</th><th>Partner</th><th>E-mail</th><th>Területi képviselő</th><th></th></tr></thead><tbody>
`
}

// CopyAssets a beágyazott képtár kimásolása.
func CopyAssets(assets fs.FS, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return fs.WalkDir(assets, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == "." {
			return err
		}
		target := filepath.Join(dst, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := fs.ReadFile(assets, p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}

// ---------------------------------------------------------------------------
// EML

// BuildEML Outlookban megnyitható, elküldetlen (X-Unsent) levelet készít.
func BuildEML(from *mail.Address, p *Partner, subject, html, text string) []byte {
	var b bytes.Buffer
	boundary := "=_energofish_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	h := func(k, v string) { b.WriteString(k + ": " + v + "\r\n") }
	h("X-Unsent", "1")
	if from != nil {
		h("From", from.String())
	}
	h("To", (&mail.Address{Name: p.Name, Address: strings.TrimSpace(p.Email)}).String())
	h("Subject", mime.QEncoding.Encode("UTF-8", subject))
	h("MIME-Version", "1.0")
	h("Content-Type", `multipart/alternative; boundary="`+boundary+`"`)
	b.WriteString("\r\n")

	b.WriteString("--" + boundary + "\r\n")
	h("Content-Type", `text/plain; charset="utf-8"`)
	h("Content-Transfer-Encoding", "quoted-printable")
	b.WriteString("\r\n")
	qp := quotedprintable.NewWriter(&b)
	_, _ = qp.Write([]byte(strings.ReplaceAll(text, "\n", "\r\n")))
	_ = qp.Close()
	b.WriteString("\r\n")

	b.WriteString("--" + boundary + "\r\n")
	h("Content-Type", `text/html; charset="utf-8"`)
	h("Content-Transfer-Encoding", "base64")
	b.WriteString("\r\n")
	enc := base64.StdEncoding.EncodeToString([]byte(html))
	for len(enc) > 76 {
		b.WriteString(enc[:76] + "\r\n")
		enc = enc[76:]
	}
	b.WriteString(enc + "\r\n")
	b.WriteString("--" + boundary + "--\r\n")
	return b.Bytes()
}

// BuildText a levél egyszerű szöveges változata (az EML második része).
func BuildText(d *RenderData, tpl *Template) string {
	v := d.Values
	var b strings.Builder
	line := func(parts ...string) {
		s := strings.TrimSpace(strings.Join(parts, " "))
		if s != "" {
			b.WriteString(s + "\n")
		}
	}
	gap := func() { b.WriteString("\n") }
	line(strings.ToUpper(v["header.title"]), "·", v["header.date"])
	if v["utility.browserLink.url"] != "" {
		line(v["utility.browserLink"]+":", v["utility.browserLink.url"])
	}
	gap()
	line(v["cover.headline"])
	line(v["cover.standfirst"])
	gap()
	line(v["note.greeting"])
	gap()
	line(v["note.body"])
	gap()
	line(v["note.signer.name"])
	line(v["note.signer.title"])
	gap()
	line(strings.ToUpper(v["offer.label"]), "·", v["offer.badge"])
	line(v["offer.intro"])
	gap()
	for _, it := range d.Items {
		line("•", it["name"], "("+it["code"]+")")
		if it["desc"] != "" {
			b.WriteString("  " + it["desc"] + "\n")
		}
		if price := strings.TrimSpace(it["price"] + "   " + it["deal"]); price != "" {
			b.WriteString("  " + price + "\n")
		}
		b.WriteString("  " + it["cta"] + ": " + it["url"] + "\n")
		gap()
	}
	if v["offer.more.url"] != "" {
		line(v["offer.more"], v["offer.more.url"])
		gap()
	}
	if tpl != nil && tpl.HasPoll && v["poll.question"] != "" {
		line(strings.ToUpper(v["poll.label"]))
		line(v["poll.question"])
		for i := 1; i <= 3; i++ {
			k := "poll.answers." + strconv.Itoa(i)
			if v[k] != "" {
				line("  ", v[k]+":", v[k+".url"])
			}
		}
		line(v["poll.note"])
		gap()
	}
	if v["rep.name"] != "" {
		line(strings.ToUpper(v["rep.label"]))
		line(v["rep.name"], func() string {
			if v["rep.region"] != "" {
				return "– " + v["rep.region"]
			}
			return ""
		}())
		line(v["rep.text"])
		if v["rep.phone.url"] != "" {
			line("Telefon:", strings.TrimPrefix(v["rep.phone.url"], "tel:"))
		}
		if v["rep.email"] != "" {
			line("E-mail:", v["rep.email"])
		}
		gap()
	}
	line("--")
	line(v["footer.company"], "·", v["footer.address"])
	line(v["footer.legal"])
	line(v["footer.unsubscribe"]+":", v["footer.unsubscribe.url"])
	line(v["footer.settings"]+":", v["footer.settings.url"])
	out := b.String()
	out = strings.ReplaceAll(out, "\u00a0", " ")
	return out
}
