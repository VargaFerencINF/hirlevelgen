package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	h "energofish/hirlevel/internal/hirlevel"
)

// Időzített küldés. A Postmark nem tud késleltetett küldést, ezért a program küld a megadott
// időpontban – ehhez futnia kell, és a gép nem lehet alvó állapotban (amíg van várakozó
// időzítés, Windowson a program nem engedi elaludni a gépet).
//
// Beütemezéskor a program ugyanazt az ellenőrzést futtatja és ugyanazt a megerősítést kéri,
// mint az éles küldés, és rögzíti a tartalmat, a termékeket, a sablont és a kijelölt partnereket:
// ha közben a következő kampányon dolgozol, az nem keveredik bele. A küldés pillanatában a
// partnertörzs frissül (a közben leiratkozók kimaradnak), a képtár feltöltődik, és minden újra
// ellenőrződik; hiba esetén nem indul küldés. Ha a program az időpontban nem futott, a küldés
// később sem indul el magától („elmaradt”).

// scheduleGrace ennyivel az időpont után még elindul a küldés (pl. a gép épp ébredt).
const scheduleGrace = 15 * time.Minute

const (
	schedWaiting   = "waiting"
	schedRunning   = "running"
	schedDone      = "done"
	schedFailed    = "failed"
	schedMissed    = "missed"
	schedCancelled = "cancelled"
)

type sendSchedule struct {
	At         time.Time   `json:"at"`
	Mode       string      `json:"mode"`
	Campaign   string      `json:"campaign"`
	Recipients []string    `json:"recipients"` // a kijelölt partnerek azonosítója (első e-mail kisbetűvel)
	Letters    int         `json:"letters"`    // a beütemezéskor ellenőrzött levelek száma
	SkipSync   bool        `json:"skipSync"`
	Template   string      `json:"template"`
	Content    h.Content   `json:"content"`
	Products   []h.Product `json:"products"`
	Created    time.Time   `json:"created"`
	Status     string      `json:"status"`
	Message    string      `json:"message,omitempty"`
	Finished   time.Time   `json:"finished,omitempty"`
}

func (a *App) scheduleFile() string { return a.postmarkFile("idozites.json") }

func (a *App) loadSchedule() *sendSchedule {
	p := a.scheduleFile()
	if p == "" {
		return nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var s sendSchedule
	if json.Unmarshal(data, &s) != nil || s.At.IsZero() {
		return nil
	}
	return &s
}

func (a *App) saveSchedule(s *sendSchedule) error {
	p := a.scheduleFile()
	if p == "" {
		return errors.New("nincs beállítások mappája")
	}
	if s == nil {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
		keepAwake(false)
		return nil
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := writeFileAtomic(p, data); err != nil {
		return err
	}
	keepAwake(s.Status == schedWaiting || s.Status == schedRunning)
	return nil
}

// scheduleView a felületnek (a rögzített tartalom és a címlista nélkül).
func scheduleView(s *sendSchedule) map[string]any {
	if s == nil {
		return nil
	}
	return map[string]any{"at": s.At, "mode": s.Mode, "modeLabel": h.SendModeLabel(s.Mode), "campaign": s.Campaign,
		"partners": len(s.Recipients), "letters": s.Letters, "template": s.Template, "products": len(h.SelectedProducts(s.Products)),
		"created": s.Created, "status": s.Status, "message": s.Message, "finished": s.Finished}
}

func (a *App) apiScheduleStatus(w http.ResponseWriter, r *http.Request) (any, error) {
	a.schedMu.Lock()
	defer a.schedMu.Unlock()
	return map[string]any{"schedule": scheduleView(a.loadSchedule())}, nil
}

// apiSchedule beütemezi a küldést (a felület előbb ellenőriz, mint az azonnali küldésnél).
func (a *App) apiSchedule(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		sendRequest
		At string `json:"at"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	at, err := time.Parse(time.RFC3339, strings.TrimSpace(req.At))
	if err != nil {
		return nil, errors.New("hibás időpont")
	}
	now := time.Now()
	if at.Before(now.Add(2 * time.Minute)) {
		return nil, errors.New("az időpont legalább 2 perccel később legyen")
	}
	if at.After(now.AddDate(0, 0, 31)) {
		return nil, errors.New("legfeljebb 31 nappal előre lehet ütemezni")
	}
	if req.Mode == h.ModeValidate {
		return nil, errors.New("a validálás nem ütemezhető – azt azonnal futtasd")
	}
	a.schedMu.Lock()
	defer a.schedMu.Unlock()
	if s := a.loadSchedule(); s != nil && (s.Status == schedWaiting || s.Status == schedRunning) {
		return nil, fmt.Errorf("már van időzített küldés (%s) – előbb töröld", s.At.Local().Format("2006.01.02. 15:04"))
	}
	prep, _, err := a.prepareSend(r.Context(), req.sendRequest, false) // a felület épp most ellenőrzött
	if err != nil {
		return nil, err
	}
	if len(prep.plan.Errors) > 0 {
		return nil, errors.New(strings.Join(prep.plan.Errors, " "))
	}
	if prep.plan.Recipients == 0 {
		return nil, errors.New("nincs kinek küldeni")
	}
	if req.Mode == h.ModeLive && req.Confirm != prep.plan.Recipients {
		return nil, fmt.Errorf("a megerősítéshez a címzettek számát (%d) kell beírni – ha közben változott, ellenőrizz újra", prep.plan.Recipients)
	}
	var ids []string
	for _, i := range prep.only {
		if i >= 0 && i < len(prep.partners) {
			if k := h.PartnerKey(&prep.partners[i]); k != "" {
				ids = append(ids, k)
			}
		}
	}
	s := &sendSchedule{At: at, Mode: req.Mode, Campaign: strings.TrimSpace(req.Campaign), Recipients: ids, Letters: prep.plan.Recipients,
		SkipSync: req.SkipSync, Template: prep.tpl.ID, Content: prep.content, Products: prep.products, Created: now, Status: schedWaiting}
	if err := a.saveSchedule(s); err != nil {
		return nil, fmt.Errorf("az időzítés nem menthető: %v", err)
	}
	log.Printf("küldés beütemezve: %s, %s, %d partner, %d levél", at.Local().Format("2006-01-02 15:04"), s.Mode, len(ids), s.Letters)
	return map[string]any{"schedule": scheduleView(s)}, nil
}

// apiScheduleClear törli a várakozó időzítést, vagy nyugtázza a befejezett / elmaradt állapotot.
func (a *App) apiScheduleClear(w http.ResponseWriter, r *http.Request) (any, error) {
	a.schedMu.Lock()
	defer a.schedMu.Unlock()
	if s := a.loadSchedule(); s != nil && s.Status == schedRunning {
		return nil, errors.New("az időzített küldés már fut – a küldés lapon állítható le")
	}
	if err := a.saveSchedule(nil); err != nil {
		return nil, err
	}
	return map[string]any{"schedule": nil}, nil
}

// StartScheduler percenként többször ellenőrzi, esedékes-e az időzített küldés.
func (a *App) StartScheduler() {
	if s := a.loadSchedule(); s != nil {
		if s.Status == schedRunning {
			// a program küldés közben állt le: a napló miatt újraindítva sem megy ki kétszer semmi
			s.Status, s.Message, s.Finished = schedFailed, "A program az időzített küldés közben leállt. A küldési napló alapján ugyanazzal a kampánynévvel újraindítva csak a hiányzó levelek mennek ki.", time.Now()
			_ = a.saveSchedule(s)
		}
		keepAwake(s.Status == schedWaiting)
	}
	go func() {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-a.quit:
				keepAwake(false)
				return
			case now := <-t.C:
				a.schedTick(now)
			}
		}
	}()
}

// schedTick: ha esedékes a várakozó időzítés, elindítja (a háttérben).
func (a *App) schedTick(now time.Time) {
	a.schedMu.Lock()
	defer a.schedMu.Unlock()
	s := a.loadSchedule()
	if s == nil || s.Status != schedWaiting || now.Before(s.At) {
		return
	}
	if now.Sub(s.At) > scheduleGrace {
		s.Status, s.Finished = schedMissed, now
		s.Message = fmt.Sprintf("A program nem futott (vagy a gép aludt) a megadott időpontban (%s), ezért a küldés nem indult el. Ellenőrizz újra, és küldd ki vagy ütemezd újra.", s.At.Local().Format("2006.01.02. 15:04"))
		_ = a.saveSchedule(s)
		log.Printf("időzített küldés elmaradt: %s", s.At.Local().Format("2006-01-02 15:04"))
		return
	}
	a.mu.Lock()
	busy := a.sendJob != nil && a.sendJob.isRunning()
	a.mu.Unlock()
	if busy {
		return // egy másik küldés még fut: a türelmi időn belül újra próbálja
	}
	s.Status, s.Message = schedRunning, "A küldés előkészítése (partnertörzs frissítése, képek feltöltése és ellenőrzése)…"
	_ = a.saveSchedule(s)
	go a.runScheduled(*s)
}

func (a *App) finishSchedule(status, msg string) {
	a.schedMu.Lock()
	defer a.schedMu.Unlock()
	s := a.loadSchedule()
	if s == nil {
		return
	}
	s.Status, s.Message, s.Finished = status, msg, time.Now()
	_ = a.saveSchedule(s)
	log.Printf("időzített küldés: %s – %s", status, msg)
}

// runScheduled az esedékes küldés: a rögzített partnerek megkeresése a mostani listában,
// teljes ellenőrzés, majd a küldés elindítása.
func (a *App) runScheduled(s sendSchedule) {
	fail := func(msg string) { a.finishSchedule(schedFailed, "A küldés nem indult el: "+msg) }
	a.mu.Lock()
	var only []int
	if a.excel != nil {
		want := map[string]bool{}
		for _, k := range s.Recipients {
			want[k] = true
		}
		for i := range a.excel.Partners {
			if want[h.PartnerKey(&a.excel.Partners[i])] {
				only = append(only, i)
			}
		}
	}
	a.mu.Unlock()
	if len(only) == 0 {
		fail("a beütemezéskor kijelölt partnerek egyike sincs a most betöltött partnerlistában (közben más lista lett betöltve?).")
		return
	}
	missing := len(s.Recipients) - len(only)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	req := sendRequest{Mode: s.Mode, Campaign: s.Campaign, Only: only, SkipSync: s.SkipSync, snap: &s}
	prep, _, early, err := a.preflight(ctx, req, s.Mode == h.ModeLive)
	switch {
	case err != nil:
		fail(err.Error())
		return
	case early != nil:
		msg, _ := early["syncFailed"].(string)
		if msg == "" {
			msg = "a partnertörzs nem frissíthető"
		}
		fail(msg + " (a régi adatokkal az időzített küldés nem indul el)")
		return
	case len(prep.plan.Errors) > 0:
		fail(strings.Join(prep.plan.Errors, " "))
		return
	case prep.plan.Recipients == 0:
		a.finishSchedule(schedDone, "Nem volt kinek küldeni (mindenki megkapta már, vagy kimaradt).")
		return
	}
	note := ""
	if missing > 0 {
		note = fmt.Sprintf(" %d beütemezett partner már nincs a listában (leiratkozott vagy kikerült).", missing)
	}
	a.schedMu.Lock()
	if cur := a.loadSchedule(); cur != nil {
		cur.Message = fmt.Sprintf("Küldés folyamatban: %d levél.%s", prep.plan.Recipients, note)
		_ = a.saveSchedule(cur)
	}
	a.schedMu.Unlock()
	a.startSendJob(prep, s.Mode, s.Campaign, func(pr h.SendProgress, err error, cancelled bool) {
		status := schedDone
		msg := fmt.Sprintf("Elküldve: %d sikeres, %d hibás, %d bizonytalan (összesen %d levél).%s", pr.OK, pr.Failed, pr.Uncertain, pr.Total, note)
		switch {
		case err != nil:
			status, msg = schedFailed, "A küldés megszakadt: "+h.MaskSecrets(err.Error())+". Ugyanazzal a kampánynévvel újraindítva csak a hiányzó levelek mennek ki."
		case cancelled:
			status, msg = schedFailed, fmt.Sprintf("A küldést leállították: %d/%d levél ment ki.", pr.Done, pr.Total)
		}
		a.finishSchedule(status, msg)
	})
}

// writeFileAtomic ideiglenes fájlba ír, majd átnevezi (félbemaradt írás nem ront el semmit).
func writeFileAtomic(p string, data []byte) error {
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
