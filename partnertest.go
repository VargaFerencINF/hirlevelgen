package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	h "energofish/hirlevel/internal/hirlevel"
)

// runPartnerTest letölti és ellenőrzi egy célcsoport élő exportját (pl. CI-ban). A forrás
// csak környezeti változóból jöhet (WEBGALAMB_TOKEN_B2B_HU…); a kimenetben csak darabszámok
// vannak – személyes adat és token nélkül.
func runPartnerTest(group string) error {
	g := h.FindB2BGroup(group)
	if g == nil {
		return errors.New("ismeretlen célcsoport: " + group)
	}
	src := h.NormalizeB2BSource(os.Getenv(g.Env))
	if src == "" {
		return fmt.Errorf("a %s környezeti változó nincs megadva", g.Env)
	}
	dir, err := os.MkdirTemp("", "partnertorzs-teszt")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	s := h.NewB2BStore(dir)
	start := time.Now()
	res, err := s.Sync(context.Background(), g.ID, src, false)
	if err != nil {
		return errors.New(h.MaskSecrets(err.Error()))
	}
	r, l := res.Report, res.Log
	log.Printf("%s: HTTP %d, %s – %d rekord, %d egyedi e-mail, %d összevont, %d kihagyott, %d token nélkül, %d képviselő nélkül, %d nem kaphat levelet",
		g.Label, l.HTTP, time.Since(start).Round(time.Millisecond), r.Records, r.Unique, r.Merged, r.Skipped, r.NoToken, r.NoRep, r.NoMail)
	if len(r.UnknownKeys) > 0 {
		log.Printf("új (ismeretlen) mezők: %s", strings.Join(r.UnknownKeys, ", "))
	}
	log.Printf("formátum: %s; duplikált e-mail cím: %d", r.Format, len(r.Duplicates))
	// a CI napló nyilvános lehet: e-mail cím és token nem kerülhet bele
	for _, w := range r.Warnings {
		log.Printf("figyelmeztetés: %s", maskEmails(h.MaskSecrets(w)))
	}
	db, _ := s.DB(g.ID)
	fc := h.B2BFacets(db, h.PartnerFilter{})
	counts := func(vals []h.FacetValue, labels bool) string {
		var parts []string
		for _, v := range vals {
			if labels {
				name := v.Value
				if name == "" {
					name = "nincs adat"
				}
				parts = append(parts, fmt.Sprintf("%s %d", name, v.Total))
			} else {
				parts = append(parts, fmt.Sprint(v.Total))
			}
		}
		return strings.Join(parts, ", ")
	}
	log.Printf("képviselők (%d): %s", len(fc.Reps), counts(fc.Reps, false))
	log.Printf("besorolás: %s", counts(fc.Levels, true))
	log.Printf("partnerbolt: %s", counts(fc.Shops, true))
	log.Printf("megyék: %d; bizományos: %d; belső másolati cím: %d", len(fc.Counties), fc.Commission["yes"], fc.Internal["yes"])
	// a megyék párosítása a partnerválasztó térképével (a megyenevek nem személyes adatok)
	if data, err := fs.ReadFile(webFS, "web/terkepek.json"); err == nil {
		if ms, err := h.LoadMaps(data); err == nil {
			var values []string
			total := map[string]int{}
			for _, v := range fc.Counties {
				values = append(values, v.Value)
				total[v.Value] = v.Total
			}
			m, match, unmatched := ms.MapMatch(g.ID, values)
			if m == nil {
				log.Printf("térkép: ehhez a célcsoporthoz nincs")
			} else {
				var lost []string
				for _, v := range unmatched {
					lost = append(lost, fmt.Sprintf("%q %d", v, total[v]))
				}
				log.Printf("térkép (%s): %d/%d megye-érték párosítva; nincs a térképen: %s", m.Title, len(match), len(values)-boolInt(hasEmpty(values)), strings.Join(lost, ", "))
			}
		}
	}
	if l.Active == 0 {
		return errors.New("egyetlen aktív partner sincs")
	}
	return nil
}

func hasEmpty(values []string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			return true
		}
	}
	return false
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

var emailInText = regexp.MustCompile(`[^\s@,;:()"']+@([^\s@,;:()"']+)`)

// maskEmails az e-mail címek helyi részét kitakarja („***@domain”).
func maskEmails(s string) string { return emailInText.ReplaceAllString(s, "***@$1") }
