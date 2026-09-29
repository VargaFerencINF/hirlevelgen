package hirlevel

// Field a közös (nem partnerenkénti) tartalom egy mezője. A lista vezérli a
// felület űrlapját, a validálást és a JSON import/exportot is.
type Field struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Group       string   `json:"group"`
	Kind        string   `json:"kind"` // text, textarea, url, image, email
	Help        string   `json:"help,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Soft        int      `json:"soft,omitempty"` // javasolt maximális hossz (karakter)
	Min         int      `json:"min,omitempty"`  // javasolt minimális hossz (karakter)
	MaxWords    int      `json:"maxWords,omitempty"`
	Only        []string `json:"only,omitempty"`   // csak ezekben a sablonokban (rövid név: v1, v2, v4)
	Tokens      bool     `json:"tokens,omitempty"` // partner-változók használhatók benne
	Half        bool     `json:"half,omitempty"`   // fél szélességű mező a felületen
}

// Group mezőcsoport a felületen.
type Group struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Desc  string   `json:"desc,omitempty"`
	Icon  string   `json:"icon"`
	Only  []string `json:"only,omitempty"`
}

var Groups = []Group{
	{ID: "alap", Title: "Alapadatok", Icon: "mail", Desc: "Tárgysor, előnézeti szöveg, fejléc és a képtár címe."},
	{ID: "borito", Title: "Borító", Icon: "image", Desc: "A nagy borítókép és a főcím blokkja."},
	{ID: "level", Title: "Személyes levél", Icon: "pen", Desc: "Megszólítás és a vezetői levél. A megszólítás partnerenként töltődik ki."},
	{ID: "ajanlat", Title: "Ajánlat", Icon: "grid", Desc: "Az ajánlatblokk szövegei. Maguk a termékek a Termékek fülön vannak."},
	{ID: "kerdes", Title: "Egy kérdés", Icon: "poll", Only: []string{"v4"}, Desc: "Csak a Partnerjelentés (v4) sablonban. Üres kérdésnél a blokk elmarad."},
	{ID: "kepviselo", Title: "Képviselő-blokk", Icon: "user", Desc: "A képviselő neve, fotója és elérhetősége az Excelből jön, itt a közös szövegek vannak."},
	{ID: "lablec", Title: "Lábléc és közösségi linkek", Icon: "footer", Desc: "Cégadatok, leiratkozás, közösségi oldalak."},
	{ID: "halado", Title: "Termék-linkek és követés", Icon: "link", Desc: "Alapértelmezett kép- és termékoldal-minták, UTM-paraméterek."},
}

var Fields = []Field{
	// --- Alapadatok ---
	{Key: "meta.subject", Label: "Tárgysor", Group: "alap", Kind: "text", Required: true, Soft: 90, Tokens: true,
		Help: "A levél tárgya a küldőben, és a HTML <title> eleme."},
	{Key: "meta.preheader", Label: "Előnézeti szöveg (preheader)", Group: "alap", Kind: "textarea", Required: true, Min: 80, Soft: 110, Tokens: true,
		Help: "A postafiókban a tárgy mellett látszó rejtett szöveg, ideálisan 80-110 karakter."},
	{Key: "meta.preheaderPoll", Label: "Preheader kiegészítés a v4 sablonhoz", Group: "alap", Kind: "text", Only: []string{"v4"}, Soft: 30,
		Help: "A Partnerjelentés (v4) sablonnál a preheader végére kerül, ha van kérdés-blokk."},
	{Key: "utility.label", Label: "Felső címke", Group: "alap", Kind: "text", Required: true, Soft: 50, Half: true,
		Help: "A levél legtetején, pl. „Partnerjelentés · 10. szám”."},
	{Key: "utility.browserLink", Label: "Webes verzió link felirata", Group: "alap", Kind: "text", Soft: 30, Half: true},
	{Key: "utility.browserLink.url", Label: "Webes verzió címe", Group: "alap", Kind: "url", Tokens: true,
		Help: "Üresen hagyva a „Megnyitás böngészőben” link nem jelenik meg."},
	{Key: "header.title", Label: "Hírlevél neve (fejléc)", Group: "alap", Kind: "text", Required: true, Soft: 30, Half: true},
	{Key: "header.date", Label: "Dátum (fejléc)", Group: "alap", Kind: "text", Required: true, Soft: 20, Half: true},
	{Key: "assets.base", Label: "Képtár (assets mappa) webcíme", Group: "alap", Kind: "url", Placeholder: "https://…/hirlevel/assets",
		Help: "A sablonok saját képeit (logó, hullámok, borító) tartalmazó, https tárhelyre feltöltött assets mappa címe, perjel nélkül. Üresen hagyva a program a kimenet mellé másolja a képeket: ez helyi megtekintéshez jó, kiküldéshez nem."},

	// --- Borító ---
	{Key: "cover.image", Label: "Borítókép", Group: "borito", Kind: "image", Required: true,
		Help: "1200×660 px JPG, 200 KB alatt, szöveg nélkül. A {assets} a képtár címére cserélődik."},
	{Key: "cover.imageAlt", Label: "Borítókép helyettesítő szövege", Group: "borito", Kind: "text", Required: true, Soft: 100,
		Help: "Kikapcsolt képeknél ez látszik."},
	{Key: "cover.kicker", Label: "Felirat a főcím felett", Group: "borito", Kind: "text", Required: true, Soft: 40},
	{Key: "cover.headline", Label: "Főcím", Group: "borito", Kind: "text", Required: true, Soft: 40, MaxWords: 6, Tokens: true,
		Help: "3-6 szó."},
	{Key: "cover.standfirst", Label: "Bevezető mondat", Group: "borito", Kind: "textarea", Required: true, Soft: 180, Tokens: true,
		Help: "Egy mondat, a fő üzenet."},
	{Key: "cover.meta.issue", Label: "Lapszám", Group: "borito", Kind: "text", Required: true, Soft: 16, Half: true},
	{Key: "cover.meta.readingTime", Label: "Olvasási idő", Group: "borito", Kind: "text", Required: true, Soft: 28, Half: true},
	{Key: "cover.meta.items", Label: "Termékszám felirat", Group: "borito", Kind: "text", Required: true, Soft: 16, Half: true, Tokens: true,
		Help: "A {termekszam} a kiválasztott termékek számára cserélődik."},
	{Key: "cover.meta.poll", Label: "Kérdés felirat", Group: "borito", Kind: "text", Soft: 16, Half: true, Only: []string{"v4"}},

	// --- Személyes levél ---
	{Key: "note.greeting", Label: "Megszólítás", Group: "level", Kind: "text", Required: true, Soft: 50, Tokens: true,
		Help: "Partnerenként töltődik ki, pl. „Kedves {nev}!”."},
	{Key: "note.greetingFallback", Label: "Megszólítás, ha a partner neve hiányzik", Group: "level", Kind: "text", Required: true, Soft: 50},
	{Key: "note.body", Label: "Levél szövege", Group: "level", Kind: "textarea", Required: true, Soft: 600, Tokens: true,
		Help: "2-3 mondat. Új sor az Enterrel."},
	{Key: "note.signer.name", Label: "Aláíró neve", Group: "level", Kind: "text", Required: true, Soft: 40, Half: true},
	{Key: "note.signer.title", Label: "Aláíró beosztása", Group: "level", Kind: "text", Required: true, Soft: 60, Half: true},
	{Key: "note.signer.portrait", Label: "Aláíró portréja", Group: "level", Kind: "image", Required: true,
		Help: "112×112 px PNG/JPG, kör alakban jelenik meg."},

	// --- Ajánlat ---
	{Key: "offer.label", Label: "Blokk címkéje", Group: "ajanlat", Kind: "text", Required: true, Soft: 40, Half: true},
	{Key: "offer.badge", Label: "Jelvény (jobbra fent)", Group: "ajanlat", Kind: "text", Required: true, Soft: 30, Half: true, Tokens: true},
	{Key: "offer.intro", Label: "Bevezető mondat", Group: "ajanlat", Kind: "textarea", Required: true, Soft: 160, Tokens: true},
	{Key: "offer.cta", Label: "Gombfelirat a termékeken", Group: "ajanlat", Kind: "text", Required: true, Soft: 12, Half: true,
		Help: "Ha a termék saját feliratot kap az Excelben, az az erősebb."},
	{Key: "offer.more", Label: "„Teljes ajánlat” link felirata", Group: "ajanlat", Kind: "text", Soft: 60},
	{Key: "offer.more.url", Label: "„Teljes ajánlat” link címe", Group: "ajanlat", Kind: "url", Tokens: true,
		Help: "Üresen hagyva a link elmarad."},

	// --- Egy kérdés (v4) ---
	{Key: "poll.label", Label: "Blokk címkéje", Group: "kerdes", Kind: "text", Only: []string{"v4"}, Soft: 40},
	{Key: "poll.question", Label: "Kérdés", Group: "kerdes", Kind: "text", Only: []string{"v4"}, Soft: 90,
		Help: "Üresen hagyva a teljes kérdés-blokk elmarad."},
	{Key: "poll.answers.1", Label: "1. válasz felirata", Group: "kerdes", Kind: "text", Only: []string{"v4"}, Soft: 40, Half: true},
	{Key: "poll.answers.1.url", Label: "1. válasz linkje", Group: "kerdes", Kind: "url", Only: []string{"v4"}, Tokens: true, Half: true},
	{Key: "poll.answers.2", Label: "2. válasz felirata", Group: "kerdes", Kind: "text", Only: []string{"v4"}, Soft: 40, Half: true},
	{Key: "poll.answers.2.url", Label: "2. válasz linkje", Group: "kerdes", Kind: "url", Only: []string{"v4"}, Tokens: true, Half: true},
	{Key: "poll.answers.3", Label: "3. válasz felirata", Group: "kerdes", Kind: "text", Only: []string{"v4"}, Soft: 40, Half: true},
	{Key: "poll.answers.3.url", Label: "3. válasz linkje", Group: "kerdes", Kind: "url", Only: []string{"v4"}, Tokens: true, Half: true},
	{Key: "poll.note", Label: "Megjegyzés a válaszok alatt", Group: "kerdes", Kind: "textarea", Only: []string{"v4"}, Soft: 140},

	// --- Képviselő-blokk ---
	{Key: "rep.label", Label: "Blokk címkéje", Group: "kepviselo", Kind: "text", Required: true, Soft: 40},
	{Key: "rep.text", Label: "Képviselő szövege", Group: "kepviselo", Kind: "textarea", Required: true, Soft: 160, Tokens: true,
		Help: "A képviselő nevében szól, minden partnernél ugyanaz."},
	{Key: "rep.phonePrefix", Label: "Telefongomb előtagja", Group: "kepviselo", Kind: "text", Soft: 4, Half: true,
		Help: "A telefonszám elé kerül a gombon, pl. ✆"},

	// --- Lábléc ---
	{Key: "footer.company", Label: "Cégnév", Group: "lablec", Kind: "text", Required: true, Soft: 40, Half: true},
	{Key: "footer.address", Label: "Székhely · cégjegyzékszám · adószám", Group: "lablec", Kind: "text", Required: true, Soft: 140},
	{Key: "footer.legal", Label: "Jogi szöveg", Group: "lablec", Kind: "textarea", Required: true, Soft: 200},
	{Key: "footer.unsubscribe", Label: "Leiratkozás felirata", Group: "lablec", Kind: "text", Required: true, Soft: 30, Half: true},
	{Key: "footer.unsubscribe.url", Label: "Leiratkozás linkje", Group: "lablec", Kind: "url", Required: true, Tokens: true, Half: true},
	{Key: "footer.settings", Label: "Beállítások felirata", Group: "lablec", Kind: "text", Required: true, Soft: 30, Half: true},
	{Key: "footer.settings.url", Label: "Beállítások linkje", Group: "lablec", Kind: "url", Required: true, Tokens: true, Half: true},
	{Key: "social.facebook.url", Label: "Facebook", Group: "lablec", Kind: "url", Half: true},
	{Key: "social.youtube.url", Label: "YouTube", Group: "lablec", Kind: "url", Half: true},
	{Key: "social.instagram.url", Label: "Instagram", Group: "lablec", Kind: "url", Half: true},
	{Key: "social.tiktok.url", Label: "TikTok", Group: "lablec", Kind: "url", Half: true},

	// --- Haladó ---
	{Key: "offer.imagePattern", Label: "Termékkép alapértelmezett címe", Group: "halado", Kind: "url",
		Help: "Ha az Excelben üres a kép link, ez lesz. A {cikkszam} a cikkszámra cserélődik."},
	{Key: "offer.urlPattern", Label: "Termékoldal alapértelmezett címe", Group: "halado", Kind: "url",
		Help: "Ha az Excelben üres a gomb link, ez lesz. A {cikkszam} a cikkszámra cserélődik."},
	{Key: "offer.linkParams", Label: "Link-kiegészítés (UTM-paraméterek)", Group: "halado", Kind: "text", Tokens: true,
		Placeholder: "utm_source=hirlevel&utm_medium=email&utm_campaign=2026-10",
		Help:        "A termék- és „teljes ajánlat” linkek végére fűzi a program. Üresen hagyva nem módosít."},
}

// FieldByKey kulcs szerinti keresés.
func FieldByKey(key string) *Field {
	for i := range Fields {
		if Fields[i].Key == key {
			return &Fields[i]
		}
	}
	return nil
}

// UsedIn megadja, hogy a mező szerepel-e az adott sablonban.
func (f *Field) UsedIn(short string) bool {
	if len(f.Only) == 0 {
		return true
	}
	for _, s := range f.Only {
		if s == short {
			return true
		}
	}
	return false
}

// IsURLKind igaz, ha a mező értéke link vagy kép címe.
func (f *Field) IsURLKind() bool { return f.Kind == "url" || f.Kind == "image" }

// TemplateInfos a három sablon leírása.
var TemplateInfos = []Template{
	{ID: "v1-sotet-lemez", Short: "v1", Name: "Sötét lemez", Desc: "Sötét fejléc, narancs csíkok, szögletes téglák."},
	{ID: "v2-waterside", Short: "v2", Name: "Waterside", Desc: "Krémszínű borító, hullámok, lekerekített téglák."},
	{ID: "v4-partnerjelentes", Short: "v4", Name: "Partnerjelentés", Desc: "Sötét borító mélységtérképpel és „Egy kérdés” blokkal.", HasPoll: true},
}

// Partner-változók (a felületen beszúrható tokenek).
type TokenInfo struct {
	Token string `json:"token"`
	Desc  string `json:"desc"`
}

var Tokens = []TokenInfo{
	{"{nev}", "Partner neve"},
	{"{email}", "Partner e-mail címe"},
	{"{ceg}", "Partner cégneve"},
	{"{kepviselo}", "Területi képviselő neve"},
	{"{kepviselo_email}", "Képviselő e-mail címe"},
	{"{kepviselo_telefon}", "Képviselő telefonszáma"},
	{"{terulet}", "Képviselő területe"},
	{"{termekszam}", "Kiválasztott termékek száma"},
	{"{assets}", "Képtár webcíme"},
}
