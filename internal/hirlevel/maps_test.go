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
		globe := m != nil && m.Kind == "globe"
		if m == nil || len(m.Regions) == 0 || (m.ViewBox == "" && !globe) {
			t.Errorf("%s: nincs térkép", g.ID)
			continue
		}
		for _, r := range m.Regions {
			if r.ID == "" || r.Label == "" || (!globe && !strings.HasPrefix(r.D, "M")) || (globe && len(r.G) < 10) {
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
		// a cseh célcsoportban szlovák partnerek is vannak
		{"B2B_CZ", "Banskobystrický kraj", "SK-BC"}, {"B2B_CZ", "Bratislavský kraj", "SK-BL"}, {"B2B_CZ", "Žilinský kraj", "SK-ZI"},
		{"B2B_CZ", "Košický kraj", "SK-KI"}, {"B2B_CZ", "Nitriansky kraj", "SK-NI"}, {"B2B_CZ", "Prešovský kraj", "SK-PV"},
		{"B2B_CZ", "Trenciansky kraj", "SK-TC"}, {"B2B_CZ", "Trenčiansky kraj", "SK-TC"}, {"B2B_CZ", "Trnavský kraj", "SK-TA"},
		{"B2B_CZ", "Zlínský kraj", "CZ-ZL"}, {"B2B_CZ", "Moravskoslezský kraj", "CZ-MO"}, {"B2B_CZ", "Kraj Vysočina", "CZ-VY"},
		{"B2B_SK", "Prešovský kraj", "SK-PV"},
		{"B2B_AT", "Wien", "AT-9"}, {"B2B_AT", "Bécs", "AT-9"}, {"B2B_AT", "Niederösterreich", "AT-3"}, {"B2B_AT", "Burgenland", "AT-1"},
		{"B2B_DE", "Bayern", "DE-BY"}, {"B2B_DE", "Bavaria", "DE-BY"}, {"B2B_DE", "Bajorország", "DE-BY"}, {"B2B_DE", "Nordrhein-Westfalen", "DE-NW"},
		{"B2B_RO", "Cluj", "RO-CJ"}, {"B2B_RO", "Județul Cluj", "RO-CJ"}, {"B2B_RO", "Hargita", "RO-HR"}, {"B2B_RO", "Harghita", "RO-HR"},
		{"B2B_RO", "Bihor", "RO-BH"}, {"B2B_RO", "București", "RO-B"},
		{"B2B_ES", "Barcelona", "ES-B"}, {"B2B_ES", "Madrid", "ES-M"},
		{"B2B_RS", "Grad Beograd", "RS-00"}, {"B2B_RS", "Severnobački okrug", "RS-01"}, {"B2B_RS", "Južno-Bački", "RS-06"},
		{"B2B_COM", "Hungary", "HU"}, {"B2B_COM", "Magyarország", "HU"}, {"B2B_COM", "Deutschland", "DE"}, {"B2B_COM", "Italy", "IT"},
		{"B2B_COM", "United States", "US"}, {"B2B_COM", "USA", "US"}, {"B2B_COM", "France", "FR"}, {"B2B_COM", "Norway", "NO"},
		{"B2B_COM", "Brazil", "BR"}, {"B2B_COM", "Japán", "JP"}, {"B2B_COM", "Australia", "AU"}, {"B2B_COM", "Croatia", "HR"},
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

// A spanyol partnertörzs „KÖZÖSSÉG - TARTOMÁNY” alakú Megye-értékei és a sima tartománynevek is.
func TestMapMatchSpain(t *testing.T) {
	m := loadTestMaps(t).Maps["B2B_ES"]
	cases := map[string]string{
		"ANDALUCÍA - ALMERÍA": "ES-AL", "ANDALUCÍA - CÁDIZ": "ES-CA", "ANDALUCÍA - CÓRDOBA": "ES-CO", "ANDALUCÍA - HUELVA": "ES-H",
		"ANDALUCÍA - MÁLAGA": "ES-MA", "ANDALUCÍA - SEVILLA": "ES-SE", "ANDALUCÍA - GRANADA": "ES-GR", "ANDALUCÍA - JAÉN": "ES-J",
		"ARAGÓN - HUESCA": "ES-HU", "ARAGÓN - ZARAGOZA": "ES-Z", "ARAGÓN - TERUEL": "ES-TE",
		"CATALUÑA - BARCELONA": "ES-B", "CATALUÑA - GIRONA": "ES-GI", "CATALUÑA - LLEIDA": "ES-L", "CATALUÑA - TARRAGONA": "ES-T",
		"COMUNIDAD VALENCIANA - VALENCIA": "ES-V", "COMUNIDAD VALENCIANA - ALICANTE": "ES-A", "COMUNIDAD VALENCIANA - CASTELLÓN": "ES-CS",
		"GALICIA - A CORUÑA": "ES-C", "GALICIA - OURENSE": "ES-OR", "GALICIA - LUGO": "ES-LU", "GALICIA - PONTEVEDRA": "ES-PO",
		"PAÍS VASCO - BIZKAIA": "ES-BI", "PAÍS VASCO - GIPUZKOA": "ES-SS", "PAÍS VASCO - ARABA/ÁLAVA": "ES-VI",
		"CASTILLA Y LEÓN - BURGOS": "ES-BU", "CASTILLA Y LEÓN - LEÓN": "ES-LE", "CASTILLA-LA MANCHA - CIUDAD REAL": "ES-CR",
		"CASTILLA-LA MANCHA - TOLEDO": "ES-TO", "EXTREMADURA - BADAJOZ": "ES-BA", "EXTREMADURA - CÁCERES": "ES-CC",
		"COMUNIDAD DE MADRID - MADRID": "ES-M", "REGIÓN DE MURCIA - MURCIA": "ES-MU", "PRINCIPADO DE ASTURIAS - ASTURIAS": "ES-O",
		"CANTABRIA - CANTABRIA": "ES-S", "LA RIOJA - LA RIOJA": "ES-LO", "NAVARRA - NAVARRA": "ES-NA",
		"ISLAS BALEARES - ISLAS BALEARES": "ES-PM", "CANARIAS - LAS PALMAS": "ES-GC", "CANARIAS - SANTA CRUZ DE TENERIFE": "ES-TF",
		"CEUTA - CEUTA": "ES-CE", "MELILLA - MELILLA": "ES-ML",
		// a javított, egységes formátum: csak a tartomány
		"Almería": "ES-AL", "Cádiz": "ES-CA", "Huesca": "ES-HU", "Lleida": "ES-L", "Lérida": "ES-L", "Girona": "ES-GI",
		"A Coruña": "ES-C", "Ourense": "ES-OR", "Bizkaia": "ES-BI", "Vizcaya": "ES-BI", "Gipuzkoa": "ES-SS", "Álava": "ES-VI",
		"Valencia": "ES-V", "Illes Balears": "ES-PM", "Las Palmas": "ES-GC", "Santa Cruz de Tenerife": "ES-TF", "Madrid": "ES-M",
	}
	for v, id := range cases {
		if ids := m.Match(v); len(ids) != 1 || ids[0] != id {
			t.Errorf("%q → %v (várt: %s)", v, ids, id)
		}
	}
	// csak közösség: az összes tartománya
	for v, n := range map[string]int{"ANDALUCÍA": 8, "Aragón": 3, "País Vasco": 3, "CATALUÑA": 4} {
		if ids := m.Match(v); len(ids) != n {
			t.Errorf("%q → %v (várt: %d tartomány)", v, ids, n)
		}
	}
}
