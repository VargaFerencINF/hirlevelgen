# Energofish hírlevél-sablonok: mezők és generátor

## Reszponzivitás: kötelező felépítés

A sablonok **stílus nélkül is reszponzívak**: a felépítés inline stílusra és hibrid (egymás alá törő) oszlopokra épül, a média-lekérdezés csak finomít. A részletes szabályok és kódminták: **`RESZPONZIV-SZABALYOK.md`**. Ez projektszabály, minden új sablon e szerint készül, és át kell mennie a `reszponziv-teszt.html` A–E próbáin. Az exportcsomag szerkezetét az `EXPORT-SZABALYOK.md` írja elő.

**A generátor ezekre a jelölésekre támaszkodik, ezeket megőrizzük:**
1. **Terméktéglák:** minden tégla `<div class="tile" …>` (a `class` az első attribútum), a téglák közvetlenül egymás után jönnek, köztük pontosan `<!--[if mso]></td><td width="177" valign="top"><![endif]-->`, sorváltásnál `<!--[if mso]></td></tr><tr><td width="177" valign="top"><![endif]-->`. A 6 tégla betűre azonos, csak a sorszám más; termékmező a rácson kívül nincs. Ebből a generátor tetszőleges termékszámú rácsot készíthet.
2. **Képviselő gombjai:** a telefon-gomb (`{{rep.phone.url}}`) és az e-mail-gomb (`{{rep.email.url}}`) külön-külön saját `<td class="rep-btn">` cellában, amely csak a saját gombja `<table>`-jét tartalmazza (`div.rep-col > table > tr > td.rep-btn > table`). Adat nélkül ez a cella rejthető.
3. **Monogram:** `<td … width="64" …>{{rep.initials}}</td>`, a cellában csak a monogram (ide kerülhet a képviselő fotója).
4. **Közösségi ikonok:** minden `{{social.…url}}` link saját `<td>` cellában, cellánként egy hálózat.
5. **Linkek:** `{{utility.browserLink.url}}` és `{{offer.more.url}}` egy-egy `<a href="{{…}}">` elem.

A sablonok csak világos megjelenésűek (`color-scheme: light only`), sötét mód nincs.

---

Küldésre kész, táblázatos e-mail sablonok, **közös mezőkulcsokkal**. Egy tartalomfájlból (JSON) mindegyik egyszerre legenerálható.

> **Az alkalmazás beépített sablonjai:** `v1-sotet-lemez` (v1), `v2-waterside` (v2), `v6-osz-avar` (v6), `v6-tel` (v6t), `v9-kiarusitas` (v9). Mindegyik e szabályok szerint készült; az alkalmazás csak kitölti őket, a HTML felépítéséhez nem nyúl. Lent a v1 csomagja a példa, a többi ugyanígy épül fel. A szabálykövetést az alkalmazás előnézetének „Telefon, stílus nélkül” és „Outlook, keskeny panel” nézete is méri.

**Csomagszerkezet** (kötelező, részletesen: `EXPORT-SZABALYOK.md`). A ZIP-ben egyetlen gyökérmappa van; a sablonfájl neve az azonosító, frissítéskor nem változik.

```
Energofish_Partner_Newsletter_Design_v1-sotet-lemez.zip
└── export-v1/
    ├── sablonok/
    │   └── v1-sotet-lemez.html   ← a generátor bemenete ({{kulcs}} helyőrzőkkel), pontosan egy fájl
    ├── elonezet/
    │   └── v1-sotet-lemez.html   ← a mintatartalommal kitöltött előnézet (helyőrző nélkül)
    ├── assets/            ← a sablon összes képe, almappa nélkül (fel kell tölteni https tárhelyre)
    │   ├── cover-balaton-1200x660.jpg
    │   ├── energofish-mark-light.png
    │   ├── portre-helyettesito.png
    │   ├── social-facebook.png
    │   ├── social-instagram.png
    │   ├── social-tiktok.png
    │   └── social-youtube.png
    ├── tartalom-v1-sotet-lemez.json   ← mintatartalom (minden kulccsal), a v1 kulcskészletével
    ├── CHANGELOG.md
    ├── MEZOK-ES-GENERATOR.md
    ├── RESZPONZIV-SZABALYOK.md   ← a kötelező reszponzív felépítés (projektszabály)
    ├── EXPORT-SZABALYOK.md       ← a kötelező ZIP-szerkezet és ellenőrzőlista (projektszabály)
    └── reszponziv-teszt.html     ← A–E próbák, mind az 5 sablon
```

---

## 1. Helyőrző-szintaxis

- Formátum: `{{kulcs.alkulcs}}`, szóköz nélkül, pontozott útvonal. Mustache- és Handlebars-kompatibilis, de egyszerű keresés-csere is elég.
- A kulcs a JSON-fában keresett útvonal: `{{offer.items.3.price}}` → `data.offer.items["3"].price`.
- **Link mezők** a JSON-ban objektumok: `{ "text": "...", "url": "..." }`.
  - `{{rep.phone}}` → a `text` érték
  - `{{rep.phone.url}}` → az `url` érték
- **Minden értéket HTML-escape-elni kell** (`& < > "`), az URL-eket is (`&` → `&amp;`).
- Üres érték megengedett (pl. `offer.items.N.deal`). A hely akkor is megmarad, így a téglák egy vonalban maradnak.
- A generátor jelezzen hibát, ha egy kulcs hiányzik a JSON-ból. Csendben üresre cserélni csak az opcionális mezőket szabad (lásd a táblázatot).

## 2. Kulcslista

✓ = a sablonban szerepel. Ahol nincs pipa, azt a sablon egyszerűen nem használja, a JSON-ban maradhat.

### Általános
| Kulcs | Típus | v1 | v2 | Megjegyzés |
|---|---|:-:|:-:|---|
| `assets.base` | URL | ✓ | ✓ | A feltöltött `assets/` mappa https címe, perjel nélkül a végén |
| `meta.subject` | szöveg | ✓ | ✓ | `<title>`, és a küldőben a tárgysor |
| `meta.preheader` | szöveg | ✓ | ✓ | Rejtett előnézeti szöveg, 80-110 karakter |
| `utility.label` | szöveg | ✓ | ✓ | pl. „Partnerjelentés · 10. szám” |
| `utility.browserLink` / `.url` | link | ✓ | ✓ | Webes verzió |
| `header.title` | szöveg | ✓ | ✓ | A hírlevél neve |
| `header.date` | szöveg | ✓ | ✓ | pl. „2026. október” |

### Borító
| Kulcs | Típus | v1 | v2 | Megjegyzés |
|---|---|:-:|:-:|---|
| `cover.image` | kép URL | ✓ | ✓ | **1200×660 JPG**, 200 KB alatt, szöveg nélkül |
| `cover.imageAlt` | szöveg | ✓ | ✓ | Kikapcsolt képeknél ez látszik |
| `cover.kicker` | szöveg | ✓ | ✓ | Kis felirat a főcím felett |
| `cover.headline` | szöveg | ✓ | ✓ | 3-6 szó |
| `cover.standfirst` | szöveg | ✓ | ✓ | Egy mondat, a fő üzenet |
| `cover.meta.issue` | szöveg | ✓ | ✓ | pl. „10. szám” |
| `cover.meta.readingTime` | szöveg | ✓ | ✓ | pl. „Olvasási idő: 2 perc” |
| `cover.meta.items` | szöveg | ✓ | ✓ | pl. „6 termék” |

### Személyes levél
| Kulcs | Típus | v1 | v2 | Megjegyzés |
|---|---|:-:|:-:|---|
| `note.greeting` | szöveg | ✓ | ✓ | Címzettenként, lásd 4. pont |
| `note.body` | szöveg | ✓ | ✓ | 2-3 mondat |
| `note.signer.portrait` | kép URL | ✓ | ✓ | 112×112 PNG/JPG, kör alakban jelenik meg |
| `note.signer.name` | szöveg | ✓ | ✓ | |
| `note.signer.title` | szöveg | ✓ | ✓ | |

### Ajánlat (6 termék, N = 1-6)
| Kulcs | Típus | v1 | v2 | Megjegyzés |
|---|---|:-:|:-:|---|
| `offer.label` | szöveg | ✓ | ✓ | pl. „01 / Októberi ajánlat” |
| `offer.badge` | szöveg | ✓ | ✓ | pl. „6 tétel · okt. 31-ig” |
| `offer.intro` | szöveg | ✓ | ✓ | Egy mondat |
| `offer.items.N.url` | URL | ✓ | ✓ | Termékoldal a B2B webshopban (kép + gomb) |
| `offer.items.N.image` | kép URL | ✓ | ✓ | Fehér hátterű, négyzetes, min. 260×260 |
| `offer.items.N.imageAlt` | szöveg | ✓ | ✓ | |
| `offer.items.N.code` | szöveg | ✓ | ✓ | Cikkszám |
| `offer.items.N.name` | szöveg | ✓ | ✓ | **max. 2 sor, kb. 26 karakter** |
| `offer.items.N.desc` | szöveg | ✓ | ✓ | **max. 2 sor, kb. 24 karakter** |
| `offer.items.N.price` | szöveg | ✓ | ✓ | pl. „3 090 Ft” |
| `offer.items.N.deal` | szöveg, opcionális | ✓ | ✓ | **1 sor, kb. 18 karakter**, üres is lehet |
| `offer.items.N.cta` | szöveg | ✓ | ✓ | Gombfelirat, pl. „Rendelés” |
| `offer.more` / `.url` | link | ✓ | ✓ | A teljes ajánlat linkje |

### Képviselő (címzettenként)
| Kulcs | Típus | v1 | v2 |
|---|---|:-:|:-:|
| `rep.label` | szöveg | ✓ | ✓ |
| `rep.initials` | szöveg (2 betű) | ✓ | ✓ |
| `rep.name`, `rep.region`, `rep.text` | szöveg | ✓ | ✓ |
| `rep.phone` / `.url` | link (`tel:+36…`) | ✓ | ✓ |
| `rep.email` / `.url` | link (`mailto:…`) | ✓ | ✓ |

### Lábléc
| Kulcs | Típus | v1 | v2 |
|---|---|:-:|:-:|
| `footer.company`, `footer.address`, `footer.legal` | szöveg | ✓ | ✓ |
| `footer.unsubscribe` / `.url` | link | ✓ | ✓ |
| `footer.settings` / `.url` | link | ✓ | ✓ |
| `social.facebook.url`, `social.youtube.url`, `social.instagram.url`, `social.tiktok.url` | URL | ✓ | ✓ |

## 3. A generátor menete

1. **Tartalom betöltése:** egy JSON a `tartalom-v1-sotet-lemez.json` szerkezetében.
2. **Validálás:**
   - kötelező kulcsok megléte (a `deal` kivételével minden);
   - hosszkorlátok (név, leírás, deal, főcím);
   - URL-ek `https://`, `tel:` vagy `mailto:` kezdettel;
   - a képek elérhetők és a megadott méretűek.
3. **Kitöltés:** minden kiválasztott sablonban a `{{kulcs}}` → escape-elt érték.
4. **Kimenet:** sablononként egy kész HTML, ez mehet a küldőrendszerbe.

Minimális Node.js példa:

```js
const fs = require('fs');
const data = JSON.parse(fs.readFileSync('tartalom.json', 'utf8'));
const esc = s => String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
const get = (o, path) => {
  let v = path.split('.').reduce((a, k) => a == null ? undefined : a[k], o);
  return v && typeof v === 'object' && 'text' in v ? v.text : v;
};
const OPTIONAL = /^offer\.items\.\d\.deal$/;

for (const name of ['v1-sotet-lemez', 'v2-waterside']) {
  const tpl = fs.readFileSync(`sablonok/${name}.html`, 'utf8');
  const missing = [];
  const html = tpl.replace(/\{\{\s*([\w.]+)\s*\}\}/g, (_, key) => {
    const v = get(data, key);
    if (v === undefined) { if (!OPTIONAL.test(key)) missing.push(key); return ''; }
    return esc(v);
  });
  if (missing.length) throw new Error(`${name}: hiányzó mezők: ${missing.join(', ')}`);
  fs.writeFileSync(`kesz/${name}.html`, html);
}
```

## 4. Személyre szabás (címzettenként)

A `note.greeting` és a `rep.*` mezők címzettenként változnak. Két lehetőség van:

- **A) A küldőrendszer tölti ki:** a JSON-ba a küldő saját merge tagjei kerülnek, és a generátor ezeket változatlanul hagyja. Az escape nem rontja el őket, mert nem tartalmaznak `& < > "` karaktert. Példa Mailchimp-szintaxissal:
  - `"greeting": "Kedves *|FNAME|*!"`, a küldőben beállított alapértékkel: „Kedves Partnerünk!”
  - `"rep": { "name": "*|REP_NAME|*", "initials": "*|REP_INIT|*", ... }`
- **B) A generátor tölti ki:** a program címzettenként lefut, egy közös tartalom-JSON és a címzett adatainak (név, képviselő) összefésülésével.

## 5. Képek és tárhely

- Az `assets/` mappát fel kell tölteni egy nyilvános https tárhelyre, a címe kerül az `assets.base` mezőbe.
- Az `assets/` tartalma (sablonelemek és mintaképek):
  - `energofish-mark-light.png`: logó a sötét fejlécben (fix mérettel hivatkozva);
  - `social-facebook/youtube/instagram/tiktok.png`: eredeti színű közösségi ikonok (72×72, 36 px-en);
  - `cover-balaton-1200x660.jpg`: minta-borítókép (mezőtartalom);
  - `portre-helyettesito.png`: helyettesítő portré (mezőtartalom).
- A borítókép mezőtartalom: kiadásonként cserélhető, mindig 1200×660-ra vágva.
- A termékképek mezőtartalmak (`offer.items.N.image`), az `images.energofish.hu` oldalról töltődnek be. A sablonban külső képhivatkozás nincs.
- A `tartalom-v1-sotet-lemez.json` relatív útvonalai (`../assets/...`) csak az `elonezet/` mappa miatt relatívak. Élesben mindenhol teljes https URL kell.

## 6. Sablonkülönbségek, amelyekre a programnak figyelnie kell

- **v2:** a borító alatti hullám a webes tervben a fotóra lógott rá. Az e-mailben ez nem megoldható, mert a borító cserélhető mezőtartalom, így itt a fotó egyenes éllel zárul. A másik két hullám képként megmaradt.
- **Outlook (asztali):**
  - a lekerekített sarkokat és a pill gombokat szögletesen mutatja;
  - a termékrács ott is 3 oszlopos marad (rejtett Outlook-táblázat);
  - mobilon (480 px alatt) a termékek egymás alatt jelennek meg, kép balra, szöveg jobbra.
- **Méret:** a kész HTML sablononként kb. 30 KB, jóval a Gmail 100 KB-os levágási határa alatt.

