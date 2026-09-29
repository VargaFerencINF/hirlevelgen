# Energofish hírlevél-sablonok: mezők és generátor

Három küldésre kész, táblázatos e-mail sablon, **közös mezőkulcsokkal**. Egy tartalomfájlból (JSON) mindhárom sablon egyszerre legenerálható.

```
export/
  sablonok/            ← a generátor bemenete ({{kulcs}} helyőrzőkkel)
    v1-sotet-lemez.html
    v2-waterside.html
    v4-partnerjelentes.html
  elonezet/            ← a mintatartalommal legenerált, böngészőben megnyitható változat
  assets/              ← a sablonok saját képei (fel kell tölteni https tárhelyre)
  tartalom-minta.json  ← teljes mintatartalom, minden kulccsal
  MEZOK-ES-GENERATOR.md
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
| Kulcs | Típus | v1 | v2 | v4 | Megjegyzés |
|---|---|:-:|:-:|:-:|---|
| `assets.base` | URL | ✓ | ✓ | ✓ | A feltöltött `assets/` mappa https címe, perjel nélkül a végén |
| `meta.subject` | szöveg | ✓ | ✓ | ✓ | `<title>`, és a küldőben a tárgysor |
| `meta.preheader` | szöveg | ✓ | ✓ | ✓ | Rejtett előnézeti szöveg, 80-110 karakter |
| `utility.label` | szöveg | ✓ | ✓ | ✓ | pl. „Partnerjelentés · 10. szám” |
| `utility.browserLink` / `.url` | link | ✓ | ✓ | ✓ | Webes verzió |
| `header.title` | szöveg | ✓ | ✓ | ✓ | A hírlevél neve |
| `header.date` | szöveg | ✓ | ✓ | ✓ | pl. „2026. október” |

### Borító
| Kulcs | Típus | v1 | v2 | v4 | Megjegyzés |
|---|---|:-:|:-:|:-:|---|
| `cover.image` | kép URL | ✓ | ✓ | ✓ | **1200×660 JPG**, 200 KB alatt, szöveg nélkül |
| `cover.imageAlt` | szöveg | ✓ | ✓ | ✓ | Kikapcsolt képeknél ez látszik |
| `cover.kicker` | szöveg | ✓ | ✓ | ✓ | Kis felirat a főcím felett |
| `cover.headline` | szöveg | ✓ | ✓ | ✓ | 3-6 szó |
| `cover.standfirst` | szöveg | ✓ | ✓ | ✓ | Egy mondat, a fő üzenet |
| `cover.meta.issue` | szöveg | ✓ | ✓ | ✓ | pl. „10. szám” |
| `cover.meta.readingTime` | szöveg | ✓ | ✓ | ✓ | pl. „Olvasási idő: 2 perc” |
| `cover.meta.items` | szöveg | ✓ | ✓ | ✓ | pl. „6 termék” |
| `cover.meta.poll` | szöveg | | | ✓ | pl. „1 kérdés” |

### Személyes levél
| Kulcs | Típus | v1 | v2 | v4 | Megjegyzés |
|---|---|:-:|:-:|:-:|---|
| `note.greeting` | szöveg | ✓ | ✓ | ✓ | Címzettenként, lásd 4. pont |
| `note.body` | szöveg | ✓ | ✓ | ✓ | 2-3 mondat |
| `note.signer.portrait` | kép URL | ✓ | ✓ | ✓ | 112×112 PNG/JPG, kör alakban jelenik meg |
| `note.signer.name` | szöveg | ✓ | ✓ | ✓ | |
| `note.signer.title` | szöveg | ✓ | ✓ | ✓ | |

### Ajánlat (6 termék, N = 1-6)
| Kulcs | Típus | v1 | v2 | v4 | Megjegyzés |
|---|---|:-:|:-:|:-:|---|
| `offer.label` | szöveg | ✓ | ✓ | ✓ | pl. „01 / Októberi ajánlat” |
| `offer.badge` | szöveg | ✓ | ✓ | ✓ | pl. „6 tétel · okt. 31-ig” |
| `offer.intro` | szöveg | ✓ | ✓ | ✓ | Egy mondat |
| `offer.items.N.url` | URL | ✓ | ✓ | ✓ | Termékoldal a B2B webshopban (kép + gomb) |
| `offer.items.N.image` | kép URL | ✓ | ✓ | ✓ | Fehér hátterű, négyzetes, min. 260×260 |
| `offer.items.N.imageAlt` | szöveg | ✓ | ✓ | ✓ | |
| `offer.items.N.code` | szöveg | ✓ | ✓ | ✓ | Cikkszám |
| `offer.items.N.name` | szöveg | ✓ | ✓ | ✓ | **max. 2 sor, kb. 26 karakter** |
| `offer.items.N.desc` | szöveg | ✓ | ✓ | ✓ | **max. 2 sor, kb. 30 karakter** |
| `offer.items.N.price` | szöveg | ✓ | ✓ | ✓ | pl. „3 090 Ft” |
| `offer.items.N.deal` | szöveg, opcionális | ✓ | ✓ | ✓ | **1 sor, kb. 18 karakter**, üres is lehet |
| `offer.items.N.cta` | szöveg | ✓ | ✓ | ✓ | Gombfelirat, pl. „Rendelés” |
| `offer.more` / `.url` | link | ✓ | ✓ | ✓ | A teljes ajánlat linkje |

### Egy kérdés (csak v4)
| Kulcs | Típus | Megjegyzés |
|---|---|---|
| `poll.label` | szöveg | pl. „02 / Egy kérdés” |
| `poll.question` | szöveg | |
| `poll.answers.1-3` / `.url` | link | Minden válasz külön link, pl. `?valasz=feeder&id=*|UNIQID|*` |
| `poll.note` | szöveg | |

### Képviselő (címzettenként)
| Kulcs | Típus | v1 | v2 | v4 |
|---|---|:-:|:-:|:-:|
| `rep.label` | szöveg | ✓ | ✓ | ✓ |
| `rep.initials` | szöveg (2 betű) | ✓ | ✓ | ✓ |
| `rep.name`, `rep.region`, `rep.text` | szöveg | ✓ | ✓ | ✓ |
| `rep.phone` / `.url` | link (`tel:+36…`) | ✓ | ✓ | ✓ |
| `rep.email` / `.url` | link (`mailto:…`) | ✓ | ✓ | ✓ |

### Lábléc
| Kulcs | Típus | v1 | v2 | v4 |
|---|---|:-:|:-:|:-:|
| `footer.company`, `footer.address`, `footer.legal` | szöveg | ✓ | ✓ | ✓ |
| `footer.unsubscribe` / `.url` | link | ✓ | ✓ | ✓ |
| `footer.settings` / `.url` | link | ✓ | ✓ | ✓ |
| `social.facebook.url`, `social.youtube.url`, `social.instagram.url`, `social.tiktok.url` | URL | ✓ | ✓ | ✓ |

## 3. A generátor menete

1. **Tartalom betöltése:** egy JSON a `tartalom-minta.json` szerkezetében.
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
const OPTIONAL = /^offer\.items\.\d\.deal$|^poll\./;

for (const name of ['v1-sotet-lemez', 'v2-waterside', 'v4-partnerjelentes']) {
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
- Az `assets/` tartalma:
  - `energofish-mark.png`, `energofish-mark-light.png`: logó (a sablon fix mérettel hivatkozik rájuk);
  - `wave-band-white.png`, `wave-white-dusk.png`: v2 hullámok;
  - `contour-band.png`: v4 mélységtérkép-sáv;
  - `cover-balaton-1200x660.jpg`: minta-borítókép;
  - `portre-helyettesito.png`: helyettesítő portré.
- A borítókép mezőtartalom: kiadásonként cserélhető, mindig 1200×660-ra vágva.
- A termék- és a közösségi ikonok az `images.energofish.hu` oldalról töltődnek be.
- A `tartalom-minta.json` relatív útvonalai (`../assets/...`) csak az `elonezet/` mappa miatt relatívak. Élesben mindenhol teljes https URL kell.

## 6. Sablonkülönbségek, amelyekre a programnak figyelnie kell

- **v4:** csak itt vannak `poll.*` és `cover.meta.poll` mezők. A v4 preheaderje végére érdemes odaírni: „Plusz egy kérdés.”
- **v2:** a borító alatti hullám a webes tervben a fotóra lógott rá. Az e-mailben ez nem megoldható, mert a borító cserélhető mezőtartalom, így itt a fotó egyenes éllel zárul. A másik két hullám képként megmaradt.
- **Outlook (asztali):**
  - a lekerekített sarkokat és a pill gombokat szögletesen mutatja;
  - a termékrács ott is 3 oszlopos marad (rejtett Outlook-táblázat);
  - mobilon 2 oszlopos a rács, 620 px alatt.
- **Méret:** a kész HTML sablononként kb. 30 KB, jóval a Gmail 100 KB-os levágási határa alatt.
