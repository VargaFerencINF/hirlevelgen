package hirlevel

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"html"
	"io"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Kiküldés Postmarkon: a partnerenkénti levelek összeállítása (ugyanaz a HTML, mint a
// generálásnál, csak feltöltött képtárral), a kizárások, a köteges küldés és a napló.

// Küldési módok.
const (
	ModeValidate = "validalas" // POSTMARK_API_TEST: a Postmark ellenőrzi a kérést, semmi nem megy ki
	ModeSandbox  = "sandbox"   // Sandbox szerver: a levelek a Postmark felületén látszanak, nem kézbesít
	ModeInternal = "belsoteszt"
	ModeLive     = "eles"
)

// ValidSendMode igaz, ha ismert küldési mód.
func ValidSendMode(m string) bool {
	return m == ModeValidate || m == ModeSandbox || m == ModeInternal || m == ModeLive
}

// SendModeLabel a mód magyar neve.
func SendModeLabel(m string) string {
	return map[string]string{ModeValidate: "Validálás", ModeSandbox: "Sandbox", ModeInternal: "Belső teszt", ModeLive: "Éles"}[m]
}

// TestUnsubscribeURL a teszt módokban a valódi leiratkozó link helyére kerül: egy belső
// tesztelő kattintása így nem iratkoztathat le egy partnert.
const TestUnsubscribeURL = "https://energofish.hu/?teszt-level-leiratkozas-letiltva"

// DefaultUTM a kiküldött levelek linkjeihez fűzött paraméterek alapértéke.
const DefaultUTM = "utm_source=partnerbrief&utm_medium=email&utm_campaign={kampany}"

// SendConfig a küldés beállításai (titkos adat nélkül).
type SendConfig struct {
	From          string   // pl. Energofish Partner Brief <hirlevel@energofish.hu>
	Stream        string   // üzenetfolyam (broadcast)
	UTM           string   // linkekhez fűzött paraméterek; {kampany} = a kampány neve
	TrackOpens    bool     // megnyitások mérése
	TrackLinks    string   // None, HtmlAndText, HtmlOnly, TextOnly
	OneClick      bool     // List-Unsubscribe-Post: egykattintásos leiratkozás
	ReplyToRep    bool     // a válasz a partner területi képviselőjéhez megy
	TestAddresses []string // belső teszt címzettjei
	TestCount     int      // belső teszt: ennyi partner levele megy minden tesztcímre
}

// SendInput a küldési terv bemenete.
type SendInput struct {
	Content    Content
	Products   []Product
	Partners   []Partner
	Only       []int
	Tpl        *Template
	Config     SendConfig
	Campaign   string
	Mode       string
	Handling   string            // a stream leiratkozás-kezelése: Custom, Postmark (üres = ismeretlen)
	Suppressed map[string]string // kisbetűs cím → ok (Postmark)
	Unsubs     map[string]bool   // kisbetűs cím → importált leiratkozott lista
	Log        *SendLog          // a kampány eddigi naplója (ismételt futtatáshoz)
}

// SendItem egy elküldendő levél.
type SendItem struct {
	PartnerID string
	Name      string
	Row       int
	Msg       PostmarkMessage
	size      int
}

// SendExcluded egy kimaradó partner vagy cím.
type SendExcluded struct {
	Row    int    `json:"row"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Reason string `json:"reason"`
}

// SendPlan az összeállított küldés.
type SendPlan struct {
	Items       []SendItem     `json:"-"`
	Recipients  int            `json:"recipients"` // levelek száma
	Excluded    []SendExcluded `json:"excluded"`
	AlreadySent int            `json:"alreadySent"`
	Uncertain   int            `json:"uncertain"`
	NoReplyTo   int            `json:"noReplyTo"`
	Big         int            `json:"big"`
	MaxKB       int            `json:"maxKB"`
	Batches     int            `json:"batches"`
	Headers     bool           `json:"headers"` // saját List-Unsubscribe fejlécek
	Errors      []string       `json:"errors"`
	Warnings    []string       `json:"warnings"`
}

// CampaignUTM a kampány linkparaméterei.
func CampaignUTM(tpl, campaign string) string {
	return strings.NewReplacer("{kampany}", url.QueryEscape(campaign), "{kampány}", url.QueryEscape(campaign),
		"{campaign}", url.QueryEscape(campaign)).Replace(strings.TrimSpace(tpl))
}

var hrefRe = regexp.MustCompile(`(?i)(\shref=")([^"]*)(")`)

// skipLink igaz, ha a linkhez nem fűzünk paramétert (leiratkozás, mailto, már van UTM).
func skipLink(u, unsub string) bool {
	l := strings.ToLower(u)
	return !(strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://")) || u == unsub ||
		strings.Contains(l, "leiratkoz") || strings.Contains(l, "unsubscribe") || strings.Contains(l, "utm_source=")
}

// AddLinkParams a HTML minden kimenő http(s) linkjéhez hozzáfűzi a paramétereket
// (a leiratkozó link, a mailto:/tel: és a már UTM-et tartalmazó linkek kivételével).
func AddLinkParams(doc, params, unsub string) string {
	if strings.TrimSpace(params) == "" {
		return doc
	}
	return hrefRe.ReplaceAllStringFunc(doc, func(m string) string {
		p := hrefRe.FindStringSubmatch(m)
		u := html.UnescapeString(p[2])
		if skipLink(u, unsub) {
			return m
		}
		return p[1] + EscapeHTML(AppendParams(u, params)) + p[3]
	})
}

var textURLRe = regexp.MustCompile(`https?://[^\s<>"]+`)

// AddTextLinkParams ugyanez a szöveges változatra.
func AddTextLinkParams(text, params, unsub string) string {
	if strings.TrimSpace(params) == "" {
		return text
	}
	return textURLRe.ReplaceAllStringFunc(text, func(u string) string {
		if skipLink(u, unsub) {
			return u
		}
		return AppendParams(u, params)
	})
}

// partnerID a partner tartós azonosítója a naplóban és a metaadatokban: az első e-mail címe
// kisbetűvel (a B2B partnertörzsben is az e-mail a kulcs).
func partnerID(p *Partner) string {
	if l := SplitEmails(p.Email); len(l) > 0 {
		return strings.ToLower(l[0])
	}
	return ""
}

// unsubscribedInExcel igaz, ha az Excel „Leiratkozott” oszlopa szerint a partner leiratkozott.
func unsubscribedInExcel(p *Partner) bool {
	for _, k := range []string{"leiratkozott", "leiratkozva", "unsubscribed"} {
		switch strings.ToLower(strings.TrimSpace(p.Extra[k])) {
		case "", "0", "nem", "no", "false", "hamis", "-":
		default:
			return true
		}
	}
	return false
}

// BuildSendPlan összeállítja a leveleket és a kizárásokat. Nem küld semmit.
func BuildSendPlan(in SendInput) *SendPlan {
	plan := &SendPlan{Excluded: []SendExcluded{}, Errors: []string{}, Warnings: []string{}}
	errf := func(f string, a ...any) { plan.Errors = append(plan.Errors, fmt.Sprintf(f, a...)) }
	warnf := func(f string, a ...any) { plan.Warnings = append(plan.Warnings, fmt.Sprintf(f, a...)) }
	cfg := in.Config
	if !ValidSendMode(in.Mode) {
		errf("ismeretlen küldési mód")
		return plan
	}
	if in.Tpl == nil {
		errf("nincs kiválasztott sablon")
		return plan
	}
	campaign := strings.TrimSpace(in.Campaign)
	if campaign == "" {
		errf("Add meg a kampány nevét.")
	}
	from, err := mail.ParseAddress(strings.TrimSpace(cfg.From))
	if err != nil {
		errf("A feladó (From) hibás: %q – pl. Energofish Partner Brief <hirlevel@energofish.hu>", cfg.From)
	}
	if strings.TrimSpace(cfg.Stream) == "" {
		errf("Add meg az üzenetfolyam (Message Stream) azonosítóját, pl. broadcast.")
	}
	base := strings.TrimSpace(in.Content["assets.base"])
	if !(strings.HasPrefix(base, "https://") || strings.HasPrefix(base, "http://")) {
		errf("A képtár webcíme (Alapadatok › Képtár webcíme) üres vagy nem webcím. Kiküldéshez a képeket fel kell tölteni egy https tárhelyre – a levelek nem tartalmazhatnak helyi vagy beágyazott (cid:) képet.")
	}
	for _, is := range append(ValidateContent(in.Content, in.Products, in.Tpl), ValidateProducts(in.Content, in.Products)...) {
		if is.Level == LevelError {
			errf("Tartalmi hiba: %s", is.Message)
		}
	}
	plan.Headers = in.Handling != "Postmark"
	switch in.Handling {
	case "Custom":
	case "Postmark":
		if in.Mode == ModeLive {
			errf("A Postmark-stream leiratkozás-kezelése még „Postmark”: élesben csak „Manage unsubscribes on your own” (Custom) beállítással küldhető, különben a Postmark a saját leiratkozó linkjét is a levelekbe teszi.")
		} else {
			warnf("A stream leiratkozás-kezelése „Postmark”: a Postmark a saját leiratkozó linkjét is a levelek aljára teszi, ezért a saját List-Unsubscribe fejlécünk most kimarad.")
		}
	default:
		if in.Mode == ModeLive {
			errf("Az üzenetfolyam beállítása nem ismert – élesen csak ellenőrzött („Custom”) streamre küldhető.")
		}
	}
	if len(plan.Errors) > 0 {
		return plan
	}

	live := in.Mode == ModeLive
	params := CampaignUTM(cfg.UTM, campaign)
	trackLinks := cfg.TrackLinks
	if trackLinks == "" {
		trackLinks = "None"
	}

	// a kiválasztott partnerek
	var list []int
	for _, i := range in.Only {
		if i >= 0 && i < len(in.Partners) {
			list = append(list, i)
		}
	}
	used := map[string]string{}
	type prepared struct {
		p       *Partner
		pid     string
		to      []string
		subject string
		html    string
		text    string
		unsub   string
	}
	var ready []prepared
	excl := func(p *Partner, email, reason string) {
		plan.Excluded = append(plan.Excluded, SendExcluded{Row: p.Row, Name: firstNonEmpty(p.Name, p.Company), Email: email, Reason: reason})
	}
	for _, idx := range list {
		p := &in.Partners[idx]
		if PartnerBlocked(p) {
			excl(p, p.Email, "hiányzó vagy érvénytelen e-mail cím")
			continue
		}
		if unsubscribedInExcel(p) {
			excl(p, p.Email, "leiratkozott (az Excel „Leiratkozott” oszlopa szerint)")
			continue
		}
		var to []string
		var reasons []string
		for _, e := range SplitEmails(p.Email) {
			l := strings.ToLower(e)
			switch {
			case in.Unsubs[l]:
				reasons = append(reasons, e+": leiratkozott (importált lista)")
			case in.Suppressed[l] != "":
				reasons = append(reasons, e+": a Postmark letiltotta – "+SuppressionReasonHU(in.Suppressed[l]))
			case used[l] != "":
				reasons = append(reasons, e+": ismétlődő cím (már szerepel: "+used[l]+")")
			default:
				used[l] = firstNonEmpty(p.Name, p.Company, e)
				to = append(to, e)
			}
		}
		if len(to) == 0 {
			excl(p, p.Email, strings.Join(reasons, "; "))
			continue
		}
		for _, r := range reasons {
			excl(p, strings.SplitN(r, ":", 2)[0], strings.TrimSpace(strings.SplitN(r, ":", 2)[1])+" – a partner többi címére megy")
		}

		d := Build(in.Content, in.Products, p, in.Tpl, "")
		unsub := strings.TrimSpace(d.Values["footer.unsubscribe.url"])
		if !(strings.HasPrefix(unsub, "https://") || strings.HasPrefix(unsub, "http://")) || strings.ContainsAny(unsub, "{} ") {
			excl(p, p.Email, "nincs érvényes leiratkozó link")
			continue
		}
		subject := strings.TrimSpace(d.Values["meta.subject"])
		if subject == "" {
			excl(p, p.Email, "üres tárgysor")
			continue
		}
		if !live {
			// teszt módban a valódi leiratkozó link nem kerülhet a levélbe (se a láthatóba, se a fejlécbe)
			unsub = TestUnsubscribeURL
			d.Values["footer.unsubscribe.url"] = unsub
		}
		body, missing := in.Tpl.Render(d)
		if len(missing) > 0 {
			errf("A sablon ismeretlen mezőket használ: %s", strings.Join(missing, ", "))
			return plan
		}
		if strings.Contains(strings.ToLower(body), `"cid:`) || strings.Contains(strings.ToLower(body), `'cid:`) {
			excl(p, p.Email, "a levélben beágyazott (cid:) kép van")
			continue
		}
		body = AddLinkParams(body, params, unsub)
		text := AddTextLinkParams(BuildText(d, in.Tpl), params, unsub)
		ready = append(ready, prepared{p: p, pid: partnerID(p), to: to, subject: subject, html: body, text: text, unsub: unsub})
	}

	// a levelek
	mk := func(pr prepared, to, subject string) PostmarkMessage {
		m := PostmarkMessage{From: from.String(), To: to, Subject: subject, HtmlBody: pr.html, TextBody: pr.text,
			MessageStream: strings.TrimSpace(cfg.Stream), Tag: campaign, TrackOpens: cfg.TrackOpens, TrackLinks: trackLinks,
			Metadata: map[string]string{"partner_id": clip(pr.pid, 80), "campaign": clip(campaign, 80)}}
		if n := strings.TrimSpace(pr.p.Extra["nazon"]); n != "" {
			m.Metadata["nazon"] = clip(n, 80)
		}
		if cfg.ReplyToRep {
			if e := strings.TrimSpace(pr.p.RepEmail); e != "" && ValidEmail(e) {
				m.ReplyTo = e
			}
		}
		if plan.Headers {
			m.Headers = []PostmarkHeader{{Name: "List-Unsubscribe", Value: "<" + pr.unsub + ">"}}
			if cfg.OneClick {
				m.Headers = append(m.Headers, PostmarkHeader{Name: "List-Unsubscribe-Post", Value: "List-Unsubscribe=One-Click"})
			}
		}
		return m
	}
	add := func(pr prepared, to, subject string) {
		key := to
		if live {
			key = ""
		}
		if in.Mode != ModeValidate && in.Log != nil {
			switch in.Log.Status(pr.pid, key) {
			case LogSent:
				plan.AlreadySent++
				return
			case LogUncertain:
				plan.Uncertain++
				return
			}
		}
		m := mk(pr, to, subject)
		if cfg.ReplyToRep && m.ReplyTo == "" {
			plan.NoReplyTo++
		}
		size := len(m.HtmlBody)
		if size > 100*1024 {
			plan.Big++
		}
		if kb := (size + 1023) / 1024; kb > plan.MaxKB {
			plan.MaxKB = kb
		}
		plan.Items = append(plan.Items, SendItem{PartnerID: pr.pid, Name: firstNonEmpty(pr.p.Name, pr.p.Company), Row: pr.p.Row, Msg: m,
			size: len(m.HtmlBody) + len(m.TextBody) + 2048})
	}
	if in.Mode == ModeInternal {
		var addrs []string
		for _, a := range cfg.TestAddresses {
			if a = strings.TrimSpace(a); a != "" {
				if !ValidEmail(a) {
					errf("Hibás belső tesztcím: %s", a)
					return plan
				}
				addrs = append(addrs, a)
			}
		}
		if len(addrs) == 0 {
			errf("A belső teszthez add meg a tesztcímeket a Postmark beállításokban.")
			return plan
		}
		n := cfg.TestCount
		if n <= 0 {
			n = 3
		}
		if n > len(ready) {
			n = len(ready)
		}
		for _, a := range addrs {
			for _, pr := range ready[:n] {
				add(pr, a, "[TESZT] "+pr.subject)
			}
		}
	} else {
		for _, pr := range ready {
			add(pr, strings.Join(pr.to, ", "), pr.subject)
		}
	}
	plan.Recipients = len(plan.Items)
	plan.Batches = len(Batches(plan.Items))
	if plan.Big > 0 {
		warnf("%d levél nagyobb 100 KB-nál (legnagyobb: %d KB) – a Gmail ezeket levágja („Az üzenet le van vágva”).", plan.Big, plan.MaxKB)
	}
	if plan.NoReplyTo > 0 {
		warnf("%d levélnél nincs képviselői e-mail cím: ezeknél a válasz a feladóhoz megy.", plan.NoReplyTo)
	}
	if plan.Uncertain > 0 {
		warnf("%d levél állapota bizonytalan egy korábbi, megszakadt küldésből – ezeket a program nem küldi újra. Nézd meg a Postmark Activity oldalán, megérkeztek-e.", plan.Uncertain)
	}
	if plan.Recipients == 0 && len(plan.Errors) == 0 {
		if plan.AlreadySent > 0 {
			errf("Ebben a kampányban és módban már mindenki megkapta a levelet (%d). Új küldéshez adj meg új kampánynevet.", plan.AlreadySent)
		} else {
			errf("Nincs kiküldhető címzett.")
		}
	}
	return plan
}

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// Batches kötegekre bontja a leveleket (legfeljebb 500 levél és 40 MB kötegenként).
func Batches(items []SendItem) [][]SendItem {
	var out [][]SendItem
	start, size := 0, 0
	for i, it := range items {
		if i > start && (i-start >= PostmarkBatchMax || size+it.size > postmarkBatchBytes) {
			out = append(out, items[start:i])
			start, size = i, 0
		}
		size += it.size
	}
	if start < len(items) {
		out = append(out, items[start:])
	}
	return out
}

// ---------------------------------------------------------------------------
// Napló: kampányonként és módonként egy CSV (UTF-8 BOM, pontosvessző). Ez az igazság forrása:
// ismételt futtatáskor a már sikeresen elküldött partnerek kimaradnak.

// Naplóbeli állapotok.
const (
	LogNone = iota
	LogSent
	LogUncertain
)

// ErrorCodeUncertain a naplóban: a köteg elküldése után megszakadt a kapcsolat.
const ErrorCodeUncertain = -1

var logHeader = []string{"partner_id", "email", "rep_email", "MessageID", "ErrorCode", "Message", "mode", "timestamp"}

// SendLog egy kampány naplója.
type SendLog struct {
	Path  string
	byPID map[string]int
	byKey map[string]int
}

// SendLogPath a napló helye: <kimeneti mappa>/postmark-naplo/<kampány>_<mód>.csv.
func SendLogPath(outDir, campaign, mode string) string {
	return filepath.Join(outDir, "postmark-naplo", SafeFileName(campaign)+"_"+mode+".csv")
}

// LoadSendLog beolvassa a naplót (ha még nincs, üres).
func LoadSendLog(path string) (*SendLog, error) {
	l := &SendLog{Path: path, byPID: map[string]int{}, byKey: map[string]int{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte("\ufeff"))))
	r.Comma = ';'
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("a küldési napló hibás (%s): %v", filepath.Base(path), err)
	}
	for i, row := range rows {
		if i == 0 && len(row) > 0 && row[0] == "partner_id" || len(row) < 5 {
			continue
		}
		code, err := strconv.Atoi(strings.TrimSpace(row[4]))
		if err != nil {
			continue
		}
		st := LogNone
		switch code {
		case 0:
			st = LogSent
		case ErrorCodeUncertain:
			st = LogUncertain
		}
		// a sikeres küldés mindent felülír; a bizonytalan csak a semmit
		pid, key := strings.ToLower(row[0]), strings.ToLower(row[0]+"|"+row[1])
		for _, m := range []struct {
			m map[string]int
			k string
		}{{l.byPID, pid}, {l.byKey, key}} {
			if st == LogSent || (st == LogUncertain && m.m[m.k] != LogSent) {
				m.m[m.k] = st
			}
		}
	}
	return l, nil
}

// Status a partner állapota a naplóban. Üres kulccsal a partnerre (éles küldés), egyébként a
// partner + címzett párra (belső teszt: minden tesztcím megkapja).
func (l *SendLog) Status(pid, to string) int {
	if l == nil {
		return LogNone
	}
	if to == "" {
		return l.byPID[strings.ToLower(pid)]
	}
	return l.byKey[strings.ToLower(pid+"|"+to)]
}

// LogRow egy naplósor.
type LogRow struct {
	PartnerID, Email, RepEmail, MessageID string
	ErrorCode                             int
	Message, Mode                         string
	Time                                  time.Time
}

// AppendLog sorokat ír a naplóba (szükség esetén fejléccel); minden köteg után lemezre ír.
func AppendLog(path string, rows []LogRow) error {
	if len(rows) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	_, statErr := os.Stat(path)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)
	w := csv.NewWriter(bw)
	w.Comma = ';'
	w.UseCRLF = true
	if errors.Is(statErr, os.ErrNotExist) {
		bw.WriteString("\ufeff")
		_ = w.Write(logHeader)
	}
	for _, r := range rows {
		_ = w.Write([]string{r.PartnerID, r.Email, r.RepEmail, r.MessageID, strconv.Itoa(r.ErrorCode), MaskSecrets(r.Message), r.Mode,
			r.Time.Format("2006-01-02 15:04:05")})
	}
	w.Flush()
	if err := bw.Flush(); err != nil {
		return err
	}
	return f.Sync()
}

// ---------------------------------------------------------------------------
// Küldés

// SendProgress a küldés állása.
type SendProgress struct {
	Total     int            `json:"total"`
	Done      int            `json:"done"`
	OK        int            `json:"ok"`
	Failed    int            `json:"failed"`
	Uncertain int            `json:"uncertain"`
	Batch     int            `json:"batch"`
	Batches   int            `json:"batches"`
	Errors    map[string]int `json:"errors"` // hibaüzenet → darab
}

// RunSend kötegenként, egymás után (egyszerre egy kapcsolattal) elküldi a leveleket, és minden
// köteg után naplóz. A cancel a köteg végén állítja meg. Hibánál (pl. érvénytelen token vagy
// bizonytalan állapot) megáll, a már naplózott levelek megmaradnak.
func RunSend(ctx context.Context, c *PostmarkClient, items []SendItem, logPath, mode string, cancel func() bool, progress func(SendProgress)) (SendProgress, error) {
	batches := Batches(items)
	pr := SendProgress{Total: len(items), Batches: len(batches), Errors: map[string]int{}}
	report := func() {
		if progress != nil {
			cp := pr
			cp.Errors = map[string]int{}
			for k, v := range pr.Errors {
				cp.Errors[k] = v
			}
			progress(cp)
		}
	}
	report()
	for bi, b := range batches {
		if cancel != nil && cancel() {
			return pr, nil
		}
		pr.Batch = bi + 1
		msgs := make([]PostmarkMessage, len(b))
		for i := range b {
			msgs[i] = b[i].Msg
		}
		now := time.Now()
		res, err := c.SendBatch(ctx, msgs)
		rows := make([]LogRow, 0, len(b))
		row := func(it SendItem, code int, msg, id string) LogRow {
			return LogRow{PartnerID: it.PartnerID, Email: it.Msg.To, RepEmail: it.Msg.ReplyTo, MessageID: id, ErrorCode: code, Message: msg, Mode: mode, Time: now}
		}
		if err != nil {
			var pe *PostmarkError
			switch {
			case errors.Is(err, ErrPostmarkUncertain):
				for _, it := range b {
					rows = append(rows, row(it, ErrorCodeUncertain, "bizonytalan: a kapcsolat a küldés közben megszakadt – ellenőrizd a Postmark Activity oldalán", ""))
				}
				pr.Uncertain += len(b)
			case errors.As(err, &pe):
				for _, it := range b {
					rows = append(rows, row(it, pe.Code, "a köteget a Postmark elutasította: "+PostmarkErrorHU(pe.Code, pe.Message), ""))
				}
				pr.Failed += len(b)
				pr.Errors[PostmarkErrorHU(pe.Code, pe.Message)] += len(b)
			default:
				// a kérés el sem ment (pl. nincs hálózat): nem naplózzuk, újra küldhető
			}
			pr.Done += len(rows)
			if lerr := AppendLog(logPath, rows); lerr != nil {
				return pr, fmt.Errorf("a napló nem írható: %v", lerr)
			}
			report()
			return pr, err
		}
		for i, it := range b {
			if i >= len(res) {
				rows = append(rows, row(it, ErrorCodeUncertain, "bizonytalan: a Postmark válaszában nincs eredmény ehhez a levélhez", ""))
				pr.Uncertain++
				continue
			}
			r := res[i]
			if r.ErrorCode == 0 {
				rows = append(rows, row(it, 0, firstNonEmpty(r.Message, "OK"), r.MessageID))
				pr.OK++
				continue
			}
			msg := PostmarkErrorHU(r.ErrorCode, r.Message)
			rows = append(rows, row(it, r.ErrorCode, msg, r.MessageID))
			pr.Failed++
			pr.Errors[PostmarkErrorHU(r.ErrorCode, "")]++
		}
		pr.Done += len(b)
		if err := AppendLog(logPath, rows); err != nil {
			return pr, fmt.Errorf("a napló nem írható: %v", err)
		}
		report()
	}
	return pr, nil
}

// ReadUnsubList egy leiratkozott-lista (CSV vagy szöveg) e-mail címei.
func ReadUnsubList(r io.Reader) []string {
	data, _ := io.ReadAll(io.LimitReader(r, 20<<20))
	re := regexp.MustCompile(`[^\s;,"'<>()\[\]]+@[^\s;,"'<>()\[\]]+\.[A-Za-z]{2,}`)
	seen := map[string]bool{}
	var out []string
	for _, m := range re.FindAllString(string(data), -1) {
		l := strings.ToLower(m)
		if !seen[l] {
			seen[l] = true
			out = append(out, l)
		}
	}
	return out
}
