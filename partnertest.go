package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
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
	for _, w := range r.Warnings {
		log.Printf("figyelmeztetés: %s", h.MaskSecrets(w))
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
	if l.Active == 0 {
		return errors.New("egyetlen aktív partner sincs")
	}
	return nil
}
