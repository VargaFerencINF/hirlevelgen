package hirlevel

import (
	"os"
	"strings"
	"testing"
)

func loadTestMaps(t *testing.T) *MapSet {
	t.Helper()
	data, err := os.ReadFile("../../web/terkepek.json")
	if err != nil {
		t.Fatal(err)
	}
	ms, err := LoadMaps(data)
	if err != nil {
		t.Fatal(err)
	}
	return ms
}

func TestMapsCoverAllGroups(t *testing.T) {
	ms := loadTestMaps(t)
	for _, g := range B2BGroups {
		m := ms.Maps[g.ID]
		if m == nil || len(m.Regions) == 0 || m.ViewBox == "" {
			t.Errorf("%s: nincs térkép", g.ID)
			continue
		}
		for _, r := range m.Regions {
			if r.ID == "" || r.Label == "" || !strings.HasPrefix(r.D, "M") {
				t.Errorf("%s: hibás régió %+v", g.ID, r.ID)
			}
			// minden régió a saját nevére egyértelműen párosul
			if ids := m.Match(r.Label); len(ids) != 1 || ids[0] != r.ID {
				t.Errorf("%s: %q → %v (várt: %s)", g.ID, r.Label, ids, r.ID)
			}
		}
	}
}

// A magyar partnertörzs mind a 20 Megye-értéke (spec2: 19 megye + Budapest) a saját megyéjére esik.
func TestMapMatchHU(t *testing.T) {
	ms := loadTestMaps(t)
	values := []string{"Pest", "Bács-Kiskun", "Szabolcs-Szatmár-Bereg", "Budapest", "Jász-Nagykun-Szolnok", "Borsod-Abaúj-Zemplén",
		"Fejér", "Békés", "Hajdú-Bihar", "Tolna", "Komárom-Esztergom", "Baranya", "Csongrád-Csanád", "Somogy", "Győr-Moson-Sopron",
		"Heves", "Veszprém", "Nógrád", "Vas", "Zala"}
	m, match, unmatched := ms.MapMatch("B2B_HU", values)
	if len(m.Regions) != 20 || len(unmatched) != 0 {
		t.Fatalf("régiók: %d, nem párosítható: %v", len(m.Regions), unmatched)
	}
	seen := map[string]string{}
	for v, ids := range match {
		if len(ids) != 1 {
			t.Errorf("%s → %v", v, ids)
			continue
		}
		if o, ok := seen[ids[0]]; ok {
			t.Errorf("%s és %s ugyanarra a régióra esik (%s)", v, o, ids[0])
		}
		seen[ids[0]] = v
	}
	for v, id := range map[string]string{"Pest megye": "HU-PE", "Csongrád": "HU-CS", "Budapest": "HU-BU", "Győr-Moson-Sopron vármegye": "HU-GS"} {
		if ids := m.Match(v); len(ids) != 1 || ids[0] != id {
			t.Errorf("%q → %v (várt: %s)", v, ids, id)
		}
	}
	if ids := m.Match("Külföld"); ids != nil {
		t.Errorf("Külföld → %v", ids)
	}
}

// Más nyelvű és más alakú megyenevek a többi országban.
func TestMapMatchOtherCountries(t *testing.T) {
	ms := loadTestMaps(t)
	cases := []struct{ group, value, id string }{
		{"B2B_SK", "Bratislavský kraj", "SK-BL"}, {"B2B_SK", "Pozsony", "SK-BL"}, {"B2B_SK", "Bratislava", "SK-BL"},
		{"B2B_SK", "Košický", "SK-KI"}, {"B2B_SK", "Kassa", "SK-KI"}, {"B2B_SK", "Nitriansky kraj", "SK-NI"},
		{"B2B_CZ", "Praha", "CZ-PR"}, {"B2B_CZ", "Hlavní město Praha", "CZ-PR"}, {"B2B_CZ", "Jihomoravský kraj", "CZ-JM"},
		{"B2B_AT", "Wien", "AT-9"}, {"B2B_AT", "Bécs", "AT-9"}, {"B2B_AT", "Niederösterreich", "AT-3"}, {"B2B_AT", "Burgenland", "AT-1"},
		{"B2B_DE", "Bayern", "DE-BY"}, {"B2B_DE", "Bavaria", "DE-BY"}, {"B2B_DE", "Bajorország", "DE-BY"}, {"B2B_DE", "Nordrhein-Westfalen", "DE-NW"},
		{"B2B_RO", "Cluj", "RO-CJ"}, {"B2B_RO", "Județul Cluj", "RO-CJ"}, {"B2B_RO", "Hargita", "RO-HR"}, {"B2B_RO", "Harghita", "RO-HR"},
		{"B2B_RO", "Bihor", "RO-BH"}, {"B2B_RO", "București", "RO-B"},
		{"B2B_ES", "Barcelona", "ES-B"}, {"B2B_ES", "Madrid", "ES-M"},
		{"B2B_RS", "Grad Beograd", "RS-00"}, {"B2B_RS", "Severnobački okrug", "RS-01"}, {"B2B_RS", "Južno-Bački", "RS-06"},
		{"B2B_COM", "Hungary", "HU"}, {"B2B_COM", "Magyarország", "HU"}, {"B2B_COM", "Deutschland", "DE"}, {"B2B_COM", "Italy", "IT"},
	}
	for _, c := range cases {
		m := ms.Maps[c.group]
		if ids := m.Match(c.value); len(ids) != 1 || ids[0] != c.id {
			t.Errorf("%s %q → %v (várt: %s)", c.group, c.value, ids, c.id)
		}
	}
	// régiócsoportok: egy név több régiót jelöl
	if ids := ms.Maps["B2B_RS"].Match("Vajdaság"); len(ids) != 7 {
		t.Errorf("Vajdaság → %v", ids)
	}
	if ids := ms.Maps["B2B_ES"].Match("Cataluña"); len(ids) != 4 {
		t.Errorf("Cataluña → %v", ids)
	}
	if ids := ms.Maps["B2B_ES"].Match("Catalunya"); len(ids) != 4 {
		t.Errorf("Catalunya → %v", ids)
	}
}

func TestMapKey(t *testing.T) {
	for in, want := range map[string]string{"Pest megye": "pest", "Bratislavský kraj": "bratislavsky", "Județul Cluj": "cluj",
		"  Győr-Moson-Sopron ": "gyormosonsopron", "Baden-Württemberg": "badenwurttemberg", "Grad Beograd": "beograd", "Megye": ""} {
		if got := MapKey(in); got != want {
			t.Errorf("MapKey(%q) = %q, várt: %q", in, got, want)
		}
	}
}
