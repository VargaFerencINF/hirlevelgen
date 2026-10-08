package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	h "energofish/hirlevel/internal/hirlevel"
)

// Küldés Postmarkon (6. lépés). A beállítások a beállítások mappájában: postmark.json (titok
// nélkül) és postmark-tokenek.dat (a tokenek, Windowson DPAPI-val titkosítva). A tokenek
// soha nem mennek ki a felületre, a naplóba vagy hibaüzenetbe – a felület csak a maszkolt alakot látja.

// PostmarkSettings a Postmark beállításai (titkos adat nélkül).
type PostmarkSettings struct {
	Stream        string   `json:"stream_id"`
	From          string   `json:"from"`
	TestAddresses []string `json:"internal_test_addresses"`
	TestCount     int      `json:"test_count"`
	Campaign      string   `json:"campaign"`
	TrackOpens    bool     `json:"track_opens"`
	TrackLinks    string   `json:"track_links"`
	UTM           string   `json:"utm"`
	OneClick      bool     `json:"one_click"`
	ReplyToRep    bool     `json:"reply_to_rep"`
}

func defaultPostmarkSettings() PostmarkSettings {
	return PostmarkSettings{Stream: "broadcast", From: "Energofish Partner Brief <hirlevel@energofish.hu>", TestCount: 3,
		TrackOpens: true, TrackLinks: "None", UTM: h.DefaultUTM, OneClick: true, ReplyToRep: true}
}

type postmarkTokens struct {
	Live    string `json:"live,omitempty"`
	Sandbox string `json:"sandbox,omitempty"`
}

const postmarkMagic = "EFPM1\n"

// sendJob egy futó (vagy befejezett) küldés.
type sendJob struct {
	mu        sync.Mutex
	Running   bool           `json:"running"`
	Mode      string         `json:"mode"`
	ModeLabel string         `json:"modeLabel"`
	Campaign  string         `json:"campaign"`
	Progress  h.SendProgress `json:"progress"`
	Error     string         `json:"lastError,omitempty"`
	Cancelled bool           `json:"cancelled"`
	LogPath   string         `json:"logPath"`
	Started   time.Time      `json:"started"`
	Finished  time.Time      `json:"finished"`
	cancelReq bool
	stop      context.CancelFunc
}

func (j *sendJob) view() map[string]any {
	j.mu.Lock()
	defer j.mu.Unlock()
	b, _ := json.Marshal(j)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	m["cancelRequested"] = j.cancelReq
	return m
}

func (a *App) postmarkFile(name string) string {
	if a.configDir == "" {
		return ""
	}
	return filepath.Join(a.configDir, name)
}

func (a *App) loadPostmarkSettings() PostmarkSettings {
	s := defaultPostmarkSettings()
	if p := a.postmarkFile("postmark.json"); p != "" {
		if data, err := os.ReadFile(p); err == nil {
			if err := json.Unmarshal(data, &s); err != nil {
				log.Printf("postmark.json hibás, alapértékek: %v", err)
				s = defaultPostmarkSettings()
			}
		}
	}
	if strings.TrimSpace(s.Stream) == "" {
		s.Stream = "broadcast"
	}
	if s.TrackLinks == "" {
		s.TrackLinks = "None"
	}
	if s.TestCount <= 0 {
		s.TestCount = 3
	}
	return s
}

func (a *App) savePostmarkSettings(s PostmarkSettings) error {
	p := a.postmarkFile("postmark.json")
	if p == "" {
		return nil
	}
	data, _ := json.MarshalIndent(s, "", "  ")
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (a *App) loadPostmarkTokens() postmarkTokens {
	var t postmarkTokens
	p := a.postmarkFile("postmark-tokenek.dat")
	if p == "" {
		return t
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return t
	}
	if bytes.HasPrefix(data, []byte(postmarkMagic)) {
		if unprotectSecret == nil {
			return t
		}
		if data, err = unprotectSecret(data[len(postmarkMagic):]); err != nil {
			log.Printf("a Postmark tokenek nem fejthetők vissza ezen a gépen/felhasználóval")
			return postmarkTokens{}
		}
	}
	_ = json.Unmarshal(data, &t)
	return t
}

func (a *App) savePostmarkTokens(t postmarkTokens) error {
	p := a.postmarkFile("postmark-tokenek.dat")
	if p == "" {
		return nil
	}
	if t.Live == "" && t.Sandbox == "" {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	data, _ := json.Marshal(t)
	if protectSecret != nil {
		enc, err := protectSecret(data)
		if err != nil {
			return fmt.Errorf("a token titkosítása nem sikerült: %v", err)
		}
		data = append([]byte(postmarkMagic), enc...)
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// unsubList az importált leiratkozott címek (kisbetűvel).
func (a *App) unsubList() map[string]bool {
	out := map[string]bool{}
	p := a.postmarkFile("leiratkozottak.txt")
	if p == "" {
		return out
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return out
	}
	for _, l := range strings.Split(string(data), "\n") {
		if l = strings.ToLower(strings.TrimSpace(l)); l != "" {
			out[l] = true
		}
	}
	return out
}

func maskedToken(t string) map[string]any {
	if t == "" {
		return map[string]any{"set": false}
	}
	return map[string]any{"set": true, "masked": h.MaskToken(t)}
}

func (a *App) postmarkView() map[string]any {
	s := a.loadPostmarkSettings()
	t := a.loadPostmarkTokens()
	a.mu.Lock()
	dir := a.state.Output.Dir
	a.mu.Unlock()
	logDir := ""
	if strings.TrimSpace(dir) != "" {
		logDir = filepath.Join(dir, "postmark-naplo")
	}
	return map[string]any{"settings": s, "live": maskedToken(t.Live), "sandbox": maskedToken(t.Sandbox),
		"unsubs": len(a.unsubList()), "protected": secretsProtected, "logDir": logDir,
		"modes": []map[string]string{
			{"id": h.ModeValidate, "label": "Validálás"}, {"id": h.ModeSandbox, "label": "Sandbox"},
			{"id": h.ModeInternal, "label": "Belső teszt"}, {"id": h.ModeLive, "label": "Éles"}}}
}

func (a *App) apiPostmarkSettings(w http.ResponseWriter, r *http.Request) (any, error) {
	return a.postmarkView(), nil
}

func (a *App) apiPostmarkSettingsSave(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Settings     *PostmarkSettings `json:"settings"`
		LiveToken    string            `json:"liveToken"`
		SandboxToken string            `json:"sandboxToken"`
		ClearLive    bool              `json:"clearLive"`
		ClearSandbox bool              `json:"clearSandbox"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if req.Settings != nil {
		s := *req.Settings
		s.Stream = strings.TrimSpace(s.Stream)
		s.From = strings.TrimSpace(s.From)
		var addrs []string
		for _, e := range s.TestAddresses {
			for _, x := range h.SplitEmails(e) {
				if !h.ValidEmail(x) {
					return nil, fmt.Errorf("hibás tesztcím: %s", x)
				}
				addrs = append(addrs, x)
			}
		}
		s.TestAddresses = addrs
		switch s.TrackLinks {
		case "None", "HtmlAndText", "HtmlOnly", "TextOnly":
		default:
			s.TrackLinks = "None"
		}
		if s.TestCount <= 0 || s.TestCount > 50 {
			s.TestCount = 3
		}
		if err := a.savePostmarkSettings(s); err != nil {
			return nil, err
		}
	}
	t := a.loadPostmarkTokens()
	changed := false
	clean := func(s string) string { return strings.Trim(strings.TrimSpace(s), "\"'<>") }
	if v := clean(req.LiveToken); v != "" {
		t.Live, changed = v, true
	}
	if v := clean(req.SandboxToken); v != "" {
		t.Sandbox, changed = v, true
	}
	if req.ClearLive {
		t.Live, changed = "", true
	}
	if req.ClearSandbox {
		t.Sandbox, changed = "", true
	}
	if changed {
		if err := a.savePostmarkTokens(t); err != nil {
			return nil, err
		}
	}
	return a.postmarkView(), nil
}

// apiPostmarkTest a kapcsolat ellenőrzése: a stream adatai és a letiltott címek száma.
func (a *App) apiPostmarkTest(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Which string `json:"which"`
	}
	_ = decode(r, &req)
	t := a.loadPostmarkTokens()
	tok := t.Live
	if req.Which == "sandbox" {
		tok = t.Sandbox
	}
	if tok == "" {
		return nil, errors.New("ehhez nincs megadva token")
	}
	s := a.loadPostmarkSettings()
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	c := a.postmarkClient(tok)
	st, err := c.Stream(ctx, s.Stream)
	if err != nil {
		return nil, err
	}
	sup, err := c.Suppressions(ctx, s.Stream)
	if err != nil {
		return nil, err
	}
	return map[string]any{"stream": streamView(st), "suppressed": len(sup)}, nil
}

// postmarkClient új kliens (tesztekben a postmarkBase címre irányítható).
func (a *App) postmarkClient(token string) *h.PostmarkClient {
	c := h.NewPostmarkClient(token)
	if a.postmarkBase != "" {
		c.BaseURL = a.postmarkBase
	} else if u := strings.TrimSpace(os.Getenv("POSTMARK_API_URL")); u != "" {
		c.BaseURL = u // fejlesztéshez: hamis Postmark szerver
	}
	if a.postmarkSleep != nil {
		c.Sleep = a.postmarkSleep
	}
	return c
}

func streamView(s *h.PostmarkStream) map[string]any {
	if s == nil {
		return nil
	}
	return map[string]any{"id": s.ID, "name": s.Name, "type": s.MessageStreamType, "handling": s.Handling()}
}

type sendRequest struct {
	Mode     string `json:"mode"`
	Campaign string `json:"campaign"`
	Only     []int  `json:"only"`
	SkipSync bool   `json:"skipSync"`
	Confirm  int    `json:"confirm"`
}

type sendPrep struct {
	plan       *h.SendPlan
	sendToken  string
	stream     *h.PostmarkStream
	suppressed int
	logPath    string
	syncInfo   map[string]any
	content    h.Content
	products   []h.Product
	partners   []h.Partner
	only       []int
	tpl        *h.Template
}

// prepareSend összeállítja a küldést: (B2B esetén) frissíti a partnertörzset, lekéri a stream
// beállítását és a letiltott címeket, beolvassa a kampány naplóját, és elkészíti a terveket.
// Ha a partnertörzs nem frissíthető, az early válasz a felületnek szól (mint a generálásnál).
func (a *App) prepareSend(ctx context.Context, req sendRequest, sync bool) (*sendPrep, map[string]any, error) {
	if !h.ValidSendMode(req.Mode) {
		return nil, nil, errors.New("válassz küldési módot")
	}
	req.Campaign = strings.TrimSpace(req.Campaign)
	if req.Campaign == "" {
		return nil, nil, errors.New("add meg a kampány nevét")
	}
	s := a.loadPostmarkSettings()
	t := a.loadPostmarkTokens()
	var sendTok, infoTok string
	switch req.Mode {
	case h.ModeValidate:
		sendTok, infoTok = h.PostmarkTestToken, t.Live
	case h.ModeSandbox:
		sendTok, infoTok = t.Sandbox, t.Sandbox
		if sendTok == "" {
			return nil, nil, errors.New("a Sandbox módhoz add meg a Sandbox szerver tokenjét a Postmark beállításokban")
		}
	default:
		sendTok, infoTok = t.Live, t.Live
		if sendTok == "" {
			return nil, nil, errors.New("add meg az éles szerver API tokenjét a Postmark beállításokban")
		}
	}
	a.mu.Lock()
	outDir := strings.TrimSpace(a.state.Output.Dir)
	hasList := a.excel != nil && len(a.excel.Partners) > 0
	a.mu.Unlock()
	if outDir == "" {
		return nil, nil, errors.New("add meg a kimeneti mappát (Generálás lépés) – oda kerül a küldési napló")
	}
	if !hasList {
		return nil, nil, errors.New("nincs betöltött partnerlista")
	}
	if len(req.Only) == 0 {
		return nil, nil, errors.New("nincs kijelölt partner")
	}
	prep := &sendPrep{sendToken: sendTok, only: req.Only}
	// B2B partnertörzs: küldés előtt kötelező frissítés (a leiratkozottak kimaradnak)
	if sync && a.excelIsB2B() && !req.SkipSync {
		only, info, err := a.refreshB2BBeforeSend(req.Only)
		if err != nil {
			if info != nil {
				return nil, info, nil
			}
			return nil, nil, err
		}
		prep.only, prep.syncInfo = only, info
	}
	a.mu.Lock()
	prep.tpl = a.template()
	prep.content = cloneContent(a.state.Content)
	prep.products = append([]h.Product{}, a.state.Products...)
	prep.partners = append([]h.Partner{}, a.excel.Partners...)
	a.mu.Unlock()

	var extraErr, extraWarn []string
	suppressed := map[string]string{}
	handling := ""
	if infoTok != "" {
		c := a.postmarkClient(infoTok)
		ictx, cancel := context.WithTimeout(ctx, 45*time.Second)
		st, err := c.Stream(ictx, s.Stream)
		switch {
		case err != nil:
			msg := fmt.Sprintf("A Postmark-stream (%s) adatai nem kérhetők le: %v", s.Stream, err)
			if req.Mode == h.ModeLive {
				extraErr = append(extraErr, msg)
			} else {
				extraWarn = append(extraWarn, msg)
			}
		default:
			prep.stream, handling = st, st.Handling()
			if st.MessageStreamType != "" && st.MessageStreamType != "Broadcasts" {
				msg := fmt.Sprintf("A(z) %q stream típusa %s, nem Broadcasts: hírlevelet a Postmark szabályai szerint broadcast streamen kell küldeni.", s.Stream, st.MessageStreamType)
				if req.Mode == h.ModeLive {
					extraErr = append(extraErr, msg)
				} else {
					extraWarn = append(extraWarn, msg)
				}
			}
		}
		if err == nil {
			sup, err := c.Suppressions(ictx, s.Stream)
			if err != nil {
				msg := fmt.Sprintf("A letiltott címek nem kérhetők le: %v", err)
				if req.Mode == h.ModeLive {
					extraErr = append(extraErr, msg)
				} else {
					extraWarn = append(extraWarn, msg)
				}
			}
			for _, x := range sup {
				switch x.SuppressionReason {
				case "HardBounce", "SpamComplaint", "ManualSuppression":
					suppressed[strings.ToLower(x.EmailAddress)] = x.SuppressionReason
				}
			}
			prep.suppressed = len(suppressed)
		}
		cancel()
	} else {
		extraWarn = append(extraWarn, "Nincs megadva éles token, ezért a stream beállítása és a letiltott címek nem ellenőrizhetők.")
	}

	prep.logPath = h.SendLogPath(outDir, req.Campaign, req.Mode)
	var sendLog *h.SendLog
	if req.Mode != h.ModeValidate {
		l, err := h.LoadSendLog(prep.logPath)
		if err != nil {
			return nil, nil, err
		}
		sendLog = l
	}
	prep.plan = h.BuildSendPlan(h.SendInput{Content: prep.content, Products: prep.products, Partners: prep.partners, Only: prep.only, Tpl: prep.tpl,
		Config: h.SendConfig{From: s.From, Stream: s.Stream, UTM: s.UTM, TrackOpens: s.TrackOpens, TrackLinks: s.TrackLinks, OneClick: s.OneClick,
			ReplyToRep: s.ReplyToRep, TestAddresses: s.TestAddresses, TestCount: s.TestCount},
		Campaign: req.Campaign, Mode: req.Mode, Handling: handling, Suppressed: suppressed, Unsubs: a.unsubList(), Log: sendLog})
	prep.plan.Errors = append(extraErr, prep.plan.Errors...)
	prep.plan.Warnings = append(extraWarn, prep.plan.Warnings...)
	if req.Mode == h.ModeInternal {
		prep.plan.Warnings = append(prep.plan.Warnings, "Belső teszt: a levelekben a valódi leiratkozó link helyén egy ártalmatlan helyettesítő link áll.")
	}
	return prep, nil, nil
}

func (a *App) sendResponse(prep *sendPrep, extra map[string]any) map[string]any {
	// a felület üres listát vár, nem null-t
	if prep.plan.Errors == nil {
		prep.plan.Errors = []string{}
	}
	if prep.plan.Warnings == nil {
		prep.plan.Warnings = []string{}
	}
	out := map[string]any{"plan": prep.plan, "stream": streamView(prep.stream), "suppressed": prep.suppressed, "logPath": prep.logPath,
		"ok": len(prep.plan.Errors) == 0}
	for k, v := range extra {
		out[k] = v
	}
	if prep.syncInfo != nil {
		a.mu.Lock()
		out["sync"], out["excel"], out["issues"] = prep.syncInfo, a.excelViewLocked(), a.issuesLocked()
		a.mu.Unlock()
	}
	return out
}

// apiPostmarkCheck a küldés előtti ellenőrzés (semmit nem küld).
func (a *App) apiPostmarkCheck(w http.ResponseWriter, r *http.Request) (any, error) {
	var req sendRequest
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	prep, early, err := a.prepareSend(r.Context(), req, true)
	if err != nil || early != nil {
		return early, err
	}
	// képek: a feltöltött képtár minden képe elérhető-e
	images := map[string]any{"checked": 0, "errors": []string{}, "warnings": 0}
	if len(prep.plan.Errors) == 0 {
		var sel []h.Partner
		for _, i := range prep.only {
			if i >= 0 && i < len(prep.partners) {
				sel = append(sel, prep.partners[i])
			}
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		check := h.CheckImages
		if a.imageCheck != nil {
			check = a.imageCheck
		}
		res := check(ctx, h.CollectImageTargets(prep.content, prep.products, sel, prep.tpl))
		cancel()
		var errs []string
		warns := 0
		for _, c := range res {
			switch c.Status {
			case "error":
				errs = append(errs, c.Where+": "+c.Message)
			case "warn":
				warns++
			}
		}
		images = map[string]any{"checked": len(res), "errors": errs, "warnings": warns}
		if len(errs) > 0 {
			msg := fmt.Sprintf("%d kép nem érhető el vagy hibás (pl. %s) – a levelekben üres hely lenne.", len(errs), errs[0])
			if req.Mode == h.ModeLive {
				prep.plan.Errors = append(prep.plan.Errors, msg)
			} else {
				prep.plan.Warnings = append(prep.plan.Warnings, msg)
			}
		}
	}
	return a.sendResponse(prep, map[string]any{"images": images}), nil
}

// apiPostmarkSend elindítja a küldést a háttérben (egyszerre csak egy futhat).
func (a *App) apiPostmarkSend(w http.ResponseWriter, r *http.Request) (any, error) {
	var req sendRequest
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	a.mu.Lock()
	running := a.sendJob != nil && a.sendJob.isRunning()
	a.mu.Unlock()
	if running {
		return nil, errors.New("már fut egy küldés")
	}
	// élesen küldés előtt is frissül a partnertörzs
	prep, early, err := a.prepareSend(r.Context(), req, req.Mode == h.ModeLive)
	if err != nil || early != nil {
		return early, err
	}
	if len(prep.plan.Errors) > 0 {
		return nil, errors.New(strings.Join(prep.plan.Errors, " "))
	}
	if req.Mode == h.ModeLive && req.Confirm != prep.plan.Recipients {
		return nil, fmt.Errorf("a megerősítéshez a címzettek számát (%d) kell beírni – ha közben változott, ellenőrizz újra", prep.plan.Recipients)
	}
	s := a.loadPostmarkSettings()
	s.Campaign = strings.TrimSpace(req.Campaign)
	_ = a.savePostmarkSettings(s)

	ctx, stop := context.WithCancel(context.Background())
	job := &sendJob{Running: true, Mode: req.Mode, ModeLabel: h.SendModeLabel(req.Mode), Campaign: s.Campaign, LogPath: prep.logPath,
		Started: time.Now(), stop: stop, Progress: h.SendProgress{Total: prep.plan.Recipients, Batches: prep.plan.Batches}}
	a.mu.Lock()
	a.sendJob = job
	a.mu.Unlock()
	client := a.postmarkClient(prep.sendToken)
	items := prep.plan.Items
	go func() {
		defer stop()
		pr, err := h.RunSend(ctx, client, items, prep.logPath, req.Mode,
			func() bool { job.mu.Lock(); defer job.mu.Unlock(); return job.cancelReq },
			func(p h.SendProgress) { job.mu.Lock(); job.Progress = p; job.mu.Unlock() })
		job.mu.Lock()
		job.Progress, job.Running, job.Finished = pr, false, time.Now()
		job.Cancelled = job.cancelReq && pr.Done < pr.Total
		if err != nil {
			job.Error = h.MaskSecrets(err.Error())
		}
		job.mu.Unlock()
		log.Printf("Postmark küldés (%s, %s): %d/%d kész, %d sikeres, %d hibás, %d bizonytalan", req.Mode, s.Campaign, pr.Done, pr.Total, pr.OK, pr.Failed, pr.Uncertain)
	}()
	return a.sendResponse(prep, map[string]any{"started": true, "job": job.view()}), nil
}

func (j *sendJob) isRunning() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.Running
}

func (a *App) apiPostmarkStatus(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	j := a.sendJob
	a.mu.Unlock()
	if j == nil {
		return map[string]any{"job": nil}, nil
	}
	return map[string]any{"job": j.view()}, nil
}

// apiPostmarkCancel: a folyamatban lévő köteg még kimegy, utána megáll.
func (a *App) apiPostmarkCancel(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	j := a.sendJob
	a.mu.Unlock()
	if j == nil {
		return map[string]any{"job": nil}, nil
	}
	j.mu.Lock()
	j.cancelReq = true
	j.mu.Unlock()
	return map[string]any{"job": j.view()}, nil
}

// apiPostmarkUnsubs leiratkozott címek importálása (CSV vagy szöveg; minden e-mail cím számít).
func (a *App) apiPostmarkUnsubs(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Text  string `json:"text"`
		Clear bool   `json:"clear"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	p := a.postmarkFile("leiratkozottak.txt")
	if p == "" {
		return nil, errors.New("nincs beállítások mappa")
	}
	if req.Clear {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return a.postmarkView(), nil
	}
	list := h.ReadUnsubList(strings.NewReader(req.Text))
	if len(list) == 0 {
		return nil, errors.New("a fájlban nincs e-mail cím")
	}
	all := a.unsubList()
	added := 0
	for _, e := range list {
		if !all[e] {
			all[e] = true
			added++
		}
	}
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	if err := os.WriteFile(p, []byte(strings.Join(keys, "\n")+"\n"), 0o600); err != nil {
		return nil, err
	}
	v := a.postmarkView()
	v["added"] = added
	return v, nil
}

// apiPostmarkLogClear egy teszt mód naplójának törlése (élesét nem lehet: az véd a dupla küldés ellen).
func (a *App) apiPostmarkLogClear(w http.ResponseWriter, r *http.Request) (any, error) {
	var req sendRequest
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if !h.ValidSendMode(req.Mode) || req.Mode == h.ModeLive {
		return nil, errors.New("az éles küldés naplója nem törölhető innen – ez védi a partnereket a dupla küldéstől")
	}
	a.mu.Lock()
	dir := a.state.Output.Dir
	a.mu.Unlock()
	if strings.TrimSpace(dir) == "" || strings.TrimSpace(req.Campaign) == "" {
		return nil, errors.New("nincs kimeneti mappa vagy kampánynév")
	}
	p := h.SendLogPath(dir, strings.TrimSpace(req.Campaign), req.Mode)
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// apiPostmarkFeedback a Postmark letiltott címei (visszapattanás, spamjelzés, leiratkozás)
// CSV-be, a betöltött partnerek adataival – az Excel / partnertörzs frissítéséhez.
func (a *App) apiPostmarkFeedback(w http.ResponseWriter, r *http.Request) (any, error) {
	t := a.loadPostmarkTokens()
	if t.Live == "" {
		return nil, errors.New("add meg az éles szerver API tokenjét a Postmark beállításokban")
	}
	s := a.loadPostmarkSettings()
	a.mu.Lock()
	dir := strings.TrimSpace(a.state.Output.Dir)
	var partners []h.Partner
	if a.excel != nil {
		partners = append(partners, a.excel.Partners...)
	}
	a.mu.Unlock()
	if dir == "" {
		return nil, errors.New("add meg a kimeneti mappát (Generálás lépés)")
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	sup, err := a.postmarkClient(t.Live).Suppressions(ctx, s.Stream)
	if err != nil {
		return nil, err
	}
	byEmail := map[string]*h.Partner{}
	for i := range partners {
		for _, e := range h.SplitEmails(partners[i].Email) {
			byEmail[strings.ToLower(e)] = &partners[i]
		}
	}
	var buf bytes.Buffer
	buf.WriteString("\ufeff")
	cw := csv.NewWriter(&buf)
	cw.Comma = ';'
	cw.UseCRLF = true
	_ = cw.Write([]string{"E-mail", "Ok", "Ok (Postmark)", "Időpont", "Eredet", "Partner", "Nazon", "Területi képviselő", "Sor a listában"})
	counts := map[string]int{}
	matched := 0
	for _, x := range sup {
		counts[x.SuppressionReason]++
		row := []string{x.EmailAddress, h.SuppressionReasonHU(x.SuppressionReason), x.SuppressionReason, x.CreatedAt, x.Origin, "", "", "", ""}
		if p := byEmail[strings.ToLower(x.EmailAddress)]; p != nil {
			matched++
			row[5], row[6], row[7], row[8] = strings.TrimSpace(p.Name+" "+p.Company), p.Extra["nazon"], p.RepName, strconv.Itoa(p.Row)
		}
		_ = cw.Write(row)
	}
	cw.Flush()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "postmark-visszajelzesek_"+time.Now().Format("2006-01-02_15-04")+".csv")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return nil, err
	}
	return map[string]any{"path": path, "total": len(sup), "hardBounce": counts["HardBounce"], "spam": counts["SpamComplaint"],
		"manual": counts["ManualSuppression"], "matched": matched}, nil
}
