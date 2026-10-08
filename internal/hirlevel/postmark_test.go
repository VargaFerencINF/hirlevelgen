package hirlevel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const pmTestToken = "11111111-2222-3333-4444-555555555555" // kitalált

func sendFixture(t *testing.T) SendInput {
	t.Helper()
	c, products := testContent(t)
	c["assets.base"] = "https://kepek.example.com/hirlevel"
	tpls, err := LoadTemplates(os.DirFS("../.."), "sablonok")
	if err != nil {
		t.Fatal(err)
	}
	partners := []Partner{
		{Row: 1, Email: "bolt1@example.com", Name: "Kiss Péter", RepName: "Nagy Attila", RepEmail: "nagy.attila@example.com",
			Unsubscribe: "https://energofish.hu/leiratkozas.html?&c=00000000000000000001", Extra: map[string]string{"nazon": "10001"}},
		{Row: 2, Email: "bolt2@example.com; Masik@Example.com", Name: "Kovács Anna", RepName: "Nagy Attila", RepEmail: "nagy.attila@example.com",
			Unsubscribe: "https://energofish.hu/leiratkozas.html?&c=00000000000000000002"},
		{Row: 3, Email: "BOLT1@example.com", Name: "Ismétlődő", Unsubscribe: "https://energofish.hu/leiratkozas.html?&c=00000000000000000003"},
		{Row: 4, Email: "visszapattant@example.com", Name: "Visszapattant", Unsubscribe: "https://energofish.hu/leiratkozas.html?&c=00000000000000000004"},
		{Row: 5, Email: "kilepett@example.com", Name: "Excelben leiratkozott", Extra: map[string]string{"leiratkozott": "igen"}},
		{Row: 6, Email: "lista@example.com", Name: "Importált listán", Unsubscribe: "https://energofish.hu/leiratkozas.html?&c=00000000000000000006"},
		{Row: 7, Email: "rossz-cim", Name: "Hibás"},
		{Row: 8, Email: "nincstk@example.com", Name: "Képviselő nélkül", Unsubscribe: "https://energofish.hu/leiratkozas.html?&c=00000000000000000008"},
	}
	return SendInput{Content: c, Products: products, Partners: partners, Only: []int{0, 1, 2, 3, 4, 5, 6, 7}, Tpl: FindTemplate(tpls, "v4"),
		Config: SendConfig{From: "Energofish Partner Brief <hirlevel@energofish.hu>", Stream: "broadcast", UTM: DefaultUTM, TrackOpens: true,
			TrackLinks: "None", OneClick: true, ReplyToRep: true, TestAddresses: []string{"teszt1@energofish.hu", "teszt2@energofish.hu"}, TestCount: 2},
		Campaign: "2026-10 őszi", Mode: ModeLive, Handling: "Custom",
		Suppressed: map[string]string{"visszapattant@example.com": "HardBounce"},
		Unsubs:     map[string]bool{"lista@example.com": true}}
}

func TestBuildSendPlanLive(t *testing.T) {
	in := sendFixture(t)
	plan := BuildSendPlan(in)
	if len(plan.Errors) > 0 {
		t.Fatalf("hibák: %v", plan.Errors)
	}
	// 1, 2 és 8 megy ki; 3 ismétlődő, 4 letiltott, 5 Excelben leiratkozott, 6 listán, 7 hibás
	if plan.Recipients != 3 || len(plan.Items) != 3 {
		t.Fatalf("címzettek: %d %+v", plan.Recipients, plan.Excluded)
	}
	reasons := map[int]string{}
	for _, e := range plan.Excluded {
		reasons[e.Row] = e.Reason
	}
	for row, want := range map[int]string{3: "ismétlődő", 4: "HardBounce", 5: "Excel", 6: "importált", 7: "érvénytelen"} {
		if !strings.Contains(reasons[row], want) {
			t.Errorf("%d. sor: %q (várt: %s)", row, reasons[row], want)
		}
	}
	if plan.NoReplyTo != 1 || plan.Batches != 1 || !plan.Headers {
		t.Errorf("terv: %+v", plan)
	}
	m := plan.Items[0].Msg
	if m.To != "bolt1@example.com" || m.ReplyTo != "nagy.attila@example.com" || m.MessageStream != "broadcast" || m.Tag != "2026-10 őszi" ||
		m.Metadata["partner_id"] != "bolt1@example.com" || m.Metadata["campaign"] != "2026-10 őszi" || m.Metadata["nazon"] != "10001" ||
		!m.TrackOpens || m.TrackLinks != "None" || !strings.HasPrefix(m.From, `"Energofish Partner Brief"`) {
		t.Errorf("levél: %+v", m)
	}
	if plan.Items[1].Msg.To != "bolt2@example.com, Masik@Example.com" {
		t.Errorf("több cím: %q", plan.Items[1].Msg.To)
	}
	// a fejlécben ugyanaz a leiratkozó link, mint a látható linkben
	unsub := in.Partners[0].Unsubscribe
	if len(m.Headers) != 2 || m.Headers[0].Value != "<"+unsub+">" || m.Headers[1].Value != "List-Unsubscribe=One-Click" {
		t.Errorf("fejlécek: %+v", m.Headers)
	}
	if !strings.Contains(m.HtmlBody, `href="`+EscapeHTML(unsub)+`"`) {
		t.Error("a látható leiratkozó link nem egyezik a fejléccel")
	}
	// UTM a kimenő linkeken, a leiratkozó linken és a mailto-n nem
	if strings.Contains(m.HtmlBody, EscapeHTML(unsub)+"&amp;utm_") || strings.Contains(m.HtmlBody, "mailto:nagy.attila@example.com?utm") ||
		strings.Contains(m.HtmlBody, "mailto:nagy.attila@example.com&amp;utm") {
		t.Error("UTM a leiratkozó vagy mailto linken")
	}
	if !strings.Contains(m.HtmlBody, "utm_campaign=2026-10+%C5%91szi") {
		t.Error("nincs UTM a linkeken")
	}
	if strings.Contains(strings.ToLower(m.HtmlBody), "cid:") || !strings.Contains(m.HtmlBody, "https://kepek.example.com/hirlevel/") {
		t.Error("a képek nem a feltöltött képtárból jönnek")
	}
	if !strings.Contains(m.TextBody, "Kedves Kiss Péter") || !strings.Contains(m.TextBody, unsub) || !strings.Contains(m.TextBody, "utm_source=partnerbrief") {
		t.Errorf("szöveges változat: %s", m.TextBody[:200])
	}
	// a partner tokene nem kerül a levélbe
	if strings.Contains(m.HtmlBody, "Token") {
		t.Error("token a levélben")
	}
}

func TestBuildSendPlanHandlingAndModes(t *testing.T) {
	in := sendFixture(t)
	// Postmark-kezelés: élesen tilos, teszt módban saját fejléc nélkül, figyelmeztetéssel
	in.Handling = "Postmark"
	if p := BuildSendPlan(in); len(p.Errors) == 0 || !strings.Contains(p.Errors[0], "Custom") {
		t.Errorf("éles + Postmark: %v", p.Errors)
	}
	in.Mode = ModeSandbox
	p := BuildSendPlan(in)
	if len(p.Errors) > 0 || p.Headers || len(p.Items[0].Msg.Headers) != 0 || len(p.Warnings) == 0 {
		t.Errorf("sandbox + Postmark: %+v", p)
	}
	// teszt módban a valódi leiratkozó link sehol nem szerepel
	for _, it := range p.Items {
		if strings.Contains(it.Msg.HtmlBody, "leiratkozas.html?") || strings.Contains(it.Msg.TextBody, "leiratkozas.html?") {
			t.Fatal("valódi leiratkozó link teszt módban")
		}
	}
	// belső teszt: a tesztcímek kapják, [TESZT] tárggyal, címenként 2 partner levele
	in.Mode, in.Handling = ModeInternal, "Custom"
	p = BuildSendPlan(in)
	if len(p.Errors) > 0 || p.Recipients != 4 {
		t.Fatalf("belső teszt: %v %d", p.Errors, p.Recipients)
	}
	for _, it := range p.Items {
		if !strings.HasPrefix(it.Msg.Subject, "[TESZT] ") || !strings.HasSuffix(it.Msg.To, "@energofish.hu") ||
			it.Msg.Headers[0].Value != "<"+TestUnsubscribeURL+">" || strings.Contains(it.Msg.HtmlBody, "leiratkozas.html?") {
			t.Errorf("tesztlevél: %s %s %v", it.Msg.To, it.Msg.Subject, it.Msg.Headers)
		}
	}
	in.Config.TestAddresses = nil
	if p := BuildSendPlan(in); len(p.Errors) == 0 {
		t.Error("tesztcím nélkül nem indulhat belső teszt")
	}
	// feltöltött képtár nélkül nem küldhető
	in = sendFixture(t)
	in.Content["assets.base"] = ""
	if p := BuildSendPlan(in); len(p.Errors) == 0 || !strings.Contains(p.Errors[0], "képtár") {
		t.Errorf("képtár nélkül: %v", p.Errors)
	}
	in = sendFixture(t)
	in.Config.UTM = ""
	if p := BuildSendPlan(in); strings.Contains(p.Items[0].Msg.HtmlBody, "utm_source=partnerbrief") {
		t.Error("üres UTM-beállításnál is került paraméter")
	}
}

func TestBatches(t *testing.T) {
	items := make([]SendItem, 1201)
	for i := range items {
		items[i].size = 1000
	}
	b := Batches(items)
	if len(b) != 3 || len(b[0]) != 500 || len(b[1]) != 500 || len(b[2]) != 201 {
		t.Errorf("kötegek: %d", len(b))
	}
	// méret szerint is bont (40 MB)
	big := make([]SendItem, 5)
	for i := range big {
		big[i].size = 15 << 20
	}
	if b := Batches(big); len(b) != 3 {
		t.Errorf("méret szerinti kötegek: %d", len(b))
	}
}

// mockPostmark egy hamis Postmark API.
type mockPostmark struct {
	mu        sync.Mutex
	fail503   int // ennyiszer válaszol 503-mal a batch kérésre
	codeFor   map[string]int
	batches   [][]PostmarkMessage
	tokens    []string
	stream    string
	supp      []PostmarkSuppression
	rejectAll bool
}

func (m *mockPostmark) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.tokens = append(m.tokens, r.Header.Get("X-Postmark-Server-Token"))
		if r.Header.Get("Accept") != "application/json" {
			t.Error("hiányzó Accept fejléc")
		}
		if r.Header.Get("X-Postmark-Server-Token") != pmTestToken {
			w.WriteHeader(401)
			fmt.Fprint(w, `{"ErrorCode":10,"Message":"Bad or missing Server API token."}`)
			return
		}
		switch {
		case r.Method == "GET" && r.URL.Path == "/message-streams/broadcast":
			fmt.Fprintf(w, `{"ID":"broadcast","Name":"Broadcasts","MessageStreamType":"Broadcasts","SubscriptionManagementConfiguration":{"UnsubscribeHandlingType":%q}}`, m.stream)
		case r.Method == "GET" && r.URL.Path == "/message-streams/broadcast/suppressions/dump":
			_ = json.NewEncoder(w).Encode(map[string]any{"Suppressions": m.supp})
		case r.Method == "POST" && r.URL.Path == "/email/batch":
			if r.Header.Get("Content-Type") != "application/json" {
				t.Error("hiányzó Content-Type")
			}
			if m.fail503 > 0 {
				m.fail503--
				w.WriteHeader(503)
				return
			}
			var msgs []PostmarkMessage
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &msgs); err != nil || len(msgs) > PostmarkBatchMax {
				w.WriteHeader(422)
				fmt.Fprint(w, `{"ErrorCode":409,"Message":"bad"}`)
				return
			}
			m.batches = append(m.batches, msgs)
			if m.rejectAll {
				w.WriteHeader(422)
				fmt.Fprint(w, `{"ErrorCode":300,"Message":"Invalid email request"}`)
				return
			}
			var out []PostmarkResult
			for i, msg := range msgs {
				code := m.codeFor[msg.To]
				res := PostmarkResult{ErrorCode: code, To: msg.To, Message: "OK", MessageID: fmt.Sprintf("id-%d-%d", len(m.batches), i)}
				if code != 0 {
					res.Message, res.MessageID = "Account is pending approval", ""
				}
				out = append(out, res)
			}
			_ = json.NewEncoder(w).Encode(out)
		default:
			w.WriteHeader(404)
		}
	}
}

func testClient(url string) (*PostmarkClient, *[]time.Duration) {
	c := NewPostmarkClient(pmTestToken)
	c.BaseURL = url
	var slept []time.Duration
	c.Sleep = func(ctx context.Context, d time.Duration) error { slept = append(slept, d); return nil }
	return c, &slept
}

func TestPostmarkStreamAndSuppressions(t *testing.T) {
	m := &mockPostmark{stream: "Custom", supp: []PostmarkSuppression{{EmailAddress: "x@example.com", SuppressionReason: "SpamComplaint"}}}
	srv := httptest.NewServer(m.handler(t))
	defer srv.Close()
	c, _ := testClient(srv.URL)
	s, err := c.Stream(context.Background(), "broadcast")
	if err != nil || s.Handling() != "Custom" || s.MessageStreamType != "Broadcasts" {
		t.Fatalf("stream: %+v %v", s, err)
	}
	sp, err := c.Suppressions(context.Background(), "broadcast")
	if err != nil || len(sp) != 1 || sp[0].SuppressionReason != "SpamComplaint" {
		t.Fatalf("letiltások: %+v %v", sp, err)
	}
	// hibás token: érthető hiba, a token nincs benne
	bad := NewPostmarkClient("rossz-token-12345678")
	bad.BaseURL = srv.URL
	_, err = bad.Stream(context.Background(), "broadcast")
	if err == nil || !strings.Contains(err.Error(), "érvénytelen vagy hiányzó API token") || strings.Contains(err.Error(), "rossz-token") {
		t.Errorf("hibás token: %v", err)
	}
}

func TestRunSendRetryErrorsAndResume(t *testing.T) {
	m := &mockPostmark{stream: "Custom", fail503: 2, codeFor: map[string]int{"bolt2@example.com, Masik@Example.com": 412}}
	srv := httptest.NewServer(m.handler(t))
	defer srv.Close()
	c, slept := testClient(srv.URL)
	dir := t.TempDir()
	logPath := SendLogPath(dir, "2026-10 őszi", ModeLive)

	in := sendFixture(t)
	plan := BuildSendPlan(in)
	var last SendProgress
	pr, err := RunSend(context.Background(), c, plan.Items, logPath, ModeLive, nil, func(p SendProgress) { last = p })
	if err != nil {
		t.Fatal(err)
	}
	// 503 kétszer → exponenciális várakozás, majd siker; a 412-es levél hibás
	if len(*slept) != 2 || (*slept)[1] != 2*(*slept)[0] {
		t.Errorf("várakozások: %v", *slept)
	}
	if pr.OK != 2 || pr.Failed != 1 || pr.Done != 3 || last.Done != 3 || pr.Errors["a fiók még teszt módban van: csak @energofish.hu címekre lehet küldeni"] != 1 {
		t.Errorf("eredmény: %+v", pr)
	}
	data, _ := os.ReadFile(logPath)
	log := string(data)
	if !strings.HasPrefix(log, "\ufeffpartner_id;email;rep_email;MessageID;ErrorCode;Message;mode;timestamp") || strings.Count(log, "\r\n") != 4 ||
		!strings.Contains(log, ";412;") || !strings.Contains(log, "id-1-0") {
		t.Errorf("napló:\n%s", log)
	}
	if strings.Contains(log, pmTestToken) {
		t.Error("token a naplóban")
	}

	// újraindítás: a sikeresen elküldöttek kimaradnak, a hibás (412) újra próbálható
	l, err := LoadSendLog(logPath)
	if err != nil {
		t.Fatal(err)
	}
	in.Log = l
	plan2 := BuildSendPlan(in)
	if plan2.AlreadySent != 2 || plan2.Recipients != 1 || plan2.Items[0].PartnerID != "bolt2@example.com" {
		t.Errorf("folytatás: %+v", plan2)
	}
	// validálás módban nincs kihagyás (semmi nem ment ki)
	in.Mode = ModeValidate
	if p := BuildSendPlan(in); p.AlreadySent != 0 || p.Recipients != 3 {
		t.Errorf("validálás: %+v", p)
	}
}

func TestRunSendBatchRejectedAndUncertain(t *testing.T) {
	m := &mockPostmark{stream: "Custom", rejectAll: true}
	srv := httptest.NewServer(m.handler(t))
	c, _ := testClient(srv.URL)
	dir := t.TempDir()
	in := sendFixture(t)
	plan := BuildSendPlan(in)
	logPath := filepath.Join(dir, "a.csv")
	pr, err := RunSend(context.Background(), c, plan.Items, logPath, ModeLive, nil, nil)
	if err == nil || pr.Failed != 3 || !strings.Contains(err.Error(), "érvénytelen levél") {
		t.Errorf("elutasított köteg: %+v %v", pr, err)
	}
	srv.Close()

	// a válasz elvész a küldés után: bizonytalan, nem küldjük újra
	hang := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer hang.Close()
	c2, _ := testClient(hang.URL)
	logPath2 := filepath.Join(dir, "b.csv")
	pr, err = RunSend(context.Background(), c2, plan.Items, logPath2, ModeLive, nil, nil)
	if err == nil || pr.Uncertain != 3 {
		t.Fatalf("bizonytalan: %+v %v", pr, err)
	}
	l, _ := LoadSendLog(logPath2)
	in.Log = l
	if p := BuildSendPlan(in); p.Uncertain != 3 || p.Recipients != 0 {
		t.Errorf("bizonytalan folytatás: %+v", p)
	}

	// megszakítás: a köteg végén áll meg
	m3 := &mockPostmark{stream: "Custom"}
	s3 := httptest.NewServer(m3.handler(t))
	defer s3.Close()
	c3, _ := testClient(s3.URL)
	items := make([]SendItem, 1100)
	for i := range items {
		items[i] = plan.Items[0]
		items[i].Msg.To = fmt.Sprintf("p%d@energofish.hu", i)
	}
	batches := 0
	pr, err = RunSend(context.Background(), c3, items, filepath.Join(dir, "c.csv"), ModeInternal, func() bool { return batches >= 1 },
		func(p SendProgress) { batches = p.Batch })
	if err != nil || pr.Done != 500 || len(m3.batches) != 1 {
		t.Errorf("megszakítás: %+v %v, kötegek: %d", pr, err, len(m3.batches))
	}
}

func TestAddLinkParams(t *testing.T) {
	doc := `<a href="https://energofish.hu/termek?a=1&amp;b=2">x</a> <a href="mailto:a@b.hu">m</a> <a href="https://energofish.hu/leiratkozas.html?&amp;c=1">l</a>` +
		` <a href="https://x.hu/?utm_source=mar">u</a> <img src="https://kep.hu/a.jpg">`
	out := AddLinkParams(doc, "utm_source=partnerbrief&utm_campaign=k", "https://energofish.hu/leiratkozas.html?&c=1")
	if !strings.Contains(out, `href="https://energofish.hu/termek?a=1&amp;b=2&amp;utm_source=partnerbrief&amp;utm_campaign=k"`) ||
		!strings.Contains(out, `href="mailto:a@b.hu"`) || !strings.Contains(out, `leiratkozas.html?&amp;c=1"`) ||
		!strings.Contains(out, `utm_source=mar"`) || !strings.Contains(out, `src="https://kep.hu/a.jpg"`) {
		t.Errorf("%s", out)
	}
	if got := ReadUnsubList(strings.NewReader("email;nev\nA@Example.com;A\n\"b@example.hu\",x\na@example.com\n")); len(got) != 2 || got[0] != "a@example.com" {
		t.Errorf("lista: %v", got)
	}
}
