# Feladat: a 4 Energofish hírlevélsablon átdolgozása „stílus nélkül is reszponzív” felépítésre

Szia! Az Energofish partnerhírlevél-sablonjain dolgozunk (a te korábbi csomagjaid: `v1-sotet-lemez`, `v2-waterside`, `v6-tel`, `v9-kiarusitas`). A sablonokat egy generátoralkalmazás tölti ki: a `{{kulcs}}` helyőrzők helyére írja a tartalmat, a termékeket és a partner adatait. **A generátor a HTML felépítéséhez nem nyúl**, ezért a levél viselkedése teljes egészében a sablonon múlik.

A sablonok ma csak akkor reszponzívak, ha a levelező végrehajtja a `<style>` blokk média-lekérdezéseit. Ez sok helyen nem teljesül. A kiküldött teszteken telefonon és keskeny Outlook-ablakban a levél 527–543 px széles marad, oldalra lóg, vagy a telefon lekicsinyíti. Ezt kell megszüntetni.

Az alábbi szabályok **kötelezők**. Kérlek, alaposan dolgozz, és minden pontot ellenőrizz minden sablonon.

---

## 1. Cél egy mondatban

**Asztali gépen 3 oszlop, telefonon 1 oszlop, és a levél soha, semmilyen levelezőben ne legyen szélesebb a képernyőnél, akkor sem, ha a levelező a teljes `<style>` blokkot eldobja.**

Az asztali megjelenés (600 px és szélesebb) maradjon pontosan olyan, mint most: a színek, motívumok, betűk, térközök, a 3 oszlopos termékrács és a sorrend is. Csak a felépítés változik, a kinézet nem.

## 2. Miért kell ez (a levelezők valós tudása, caniemail.com, 2026. szeptember)

| Levelező | `<style>` / `@media` | `calc()` | Mit jelent nekünk |
|---|---|---|---|
| Apple Mail (iPhone, Mac), Samsung Email | ✅ / ✅ | ✅ | a média-lekérdezés működik |
| Gmail app **Google-fiókkal**, Gmail web | ✅ / ✅ | ✅ | működik |
| Gmail app **nem Google-fiókkal** (pl. céges IMAP/Exchange postafiók) | ❌ / ❌ (a teljes `<style>` eltűnik) | ❌ | **csak az inline stílus számít** |
| Thunderbird | ✅ / ❌ (nincs média-lekérdezés) | ✅ | csak az inline stílus és a calc számít |
| Outlook.com, új Outlook (Windows), Outlook web | ✅ / ✅, de a **teljes böngészőablakhoz** méri, nem az olvasópanelhez | ❌ | keskeny olvasópanelben a média-lekérdezés nem lép életbe |
| Outlook iOS / Android app | részleges | ❌ | a calc nem működik, a média-lekérdezés nem megbízható |
| Klasszikus Outlook (Windows, Word-motor) | csak részben | ❌ | fix 600 px a rejtett `<!--[if mso]>` táblázatokkal; ez marad így |
| Yahoo | ✅ / részleges | ❌ | — |

**Következmény:** a reszponzivitás alapját **inline stílussal**, a tartalomhoz igazodó („hibrid”) felépítéssel kell megoldani. A média-lekérdezés csak finomíthat (pl. a kártyás termékelrendezés), de **nélküle sem lóghat ki semmi**.

## 3. A rétegek (minden sablonban így)

1. **Alapréteg, csak inline stílus, ennek egyedül is működnie kell:** minden egymás melletti elem `display:inline-block` blokk, fix pixeles **legnagyobb** szélességgel. Ha nem fér el, magától új sorba kerül. Képek: `width:100%; max-width:…px; height:auto`.
2. **calc-réteg (inline):** a terméktéglák a „Fab Four” technikával teljes szélességűek lesznek, ha a rács 480 px-nél keskenyebb (lásd 5. pont). Ahol a calc nem működik, ott az 1. réteg marad érvényben.
3. **Média-lekérdezés (`<style>`):** csak finomítás, például a meglévő 480 px alatti kártyás terméktégla (kép balra 96 px, szöveg jobbra), a nagyobb betűk és a kisebb belső margók.
4. **Klasszikus Outlook:** rejtett `<!--[if mso]>` táblázat („ghost table”) minden többoszlopos sorhoz, fix szélességekkel, mint most a termékrácsnál.

## 4. Kötelező szabályok

**4.1. Külső keret** (ez jó a mostani sablonokban is, maradjon): a külső táblázat `width="100%"`, a levéltest `width="600"` attribútummal és `style="width:100%;max-width:600px;"` stílussal, előtte `<!--[if mso]><table … width="600" align="center"><tr><td><![endif]-->`.

**4.2. Egymás melletti oszlopok sehol nem lehetnek sima, egymás melletti `<td>` cellák**, ha a tartalmuk együtt 280 px-nél szélesebb lehet. Ilyen sor jelenleg:
- a fejléc felső sora („Partnerjelentés · 10. szám” és „Megnyitás böngészőben”);
- a logó és a dátum sora;
- a képviselő telefon- és e-mail-gombja;
- a v9-ben a háromcellás mezősor (`.mcell`), valamint a bevezető és a bélyegző sora;
- a v6 Tél és a v9 lábléc-sorai, ha két oszloposak;
- és minden hasonló sor.

Mindegyiket hibrid oszlopokra kell átírni:

```html
<div style="font-size:0;line-height:0;text-align:left;">
<!--[if mso]><table role="presentation" cellpadding="0" cellspacing="0" border="0" width="536"><tr><td width="268" valign="top"><![endif]-->
<div style="display:inline-block;vertical-align:top;width:100%;max-width:268px;font-size:16px;line-height:24px;">
  … bal oldal (saját táblázattal, width="100%") …
</div><!--[if mso]></td><td width="268" valign="top" align="right"><![endif]-->
<div style="display:inline-block;vertical-align:top;width:100%;max-width:268px;font-size:16px;line-height:24px;">
  … jobb oldal …
</div>
<!--[if mso]></td></tr></table><![endif]-->
</div>
```

- A szülőn `font-size:0`, hogy a blokkok közé ne kerüljön hézag. A blokkokon a betűméretet és a sormagasságot vissza kell állítani.
- A `max-width` értékek összege nem lehet nagyobb a tartalom szélességénél (a 600 px-es levélben a belső margók után, pl. 536 vagy 532 px).
- Ha asztalon az egyik oldal jobbra igazított (pl. „Megnyitás böngészőben”), ezt a blokkon belüli táblázat `align="right"` / `text-align:right` beállítása adja. Mobilon (`@media`) balra igazodhat.
- A 480 px alatti média-lekérdezés ezeket a blokkokat `width:100% !important; max-width:100% !important;` értékre állíthatja.

**4.3. Termékrács, pontosan ez a minta, a mostani téglatartalommal:**

```html
<div style="font-size:0;line-height:0;text-align:left;">
<!--[if mso]><table role="presentation" cellpadding="0" cellspacing="0" border="0" width="532"><tr><td width="177" valign="top"><![endif]-->
<div class="tile" style="display:inline-block;vertical-align:top;width:177px;min-width:177px;max-width:100%;width:calc(230400px - 48000%);">
  … a tégla mostani tartalma (t-gap, t-card stb.), {{offer.items.1.…}} …
</div><!--[if mso]></td><td width="177" valign="top"><![endif]-->
<div class="tile" style="…ugyanaz…"> … {{offer.items.2.…}} … </div><!--[if mso]></td><td width="177" valign="top"><![endif]-->
<div class="tile" style="…ugyanaz…"> … {{offer.items.3.…}} … </div><!--[if mso]></td></tr><tr><td width="177" valign="top"><![endif]-->
<div class="tile" style="…ugyanaz…"> … {{offer.items.4.…}} … </div><!--[if mso]></td><td width="177" valign="top"><![endif]-->
… 5., 6. …
<!--[if mso]></td></tr></table><![endif]-->
</div>
```

A tégla stílusa két réteg:
- `width:177px; min-width:177px; max-width:100%` az alap. Asztalon 3 × 177 = 531 px egy sorban, keskeny helyen a téglák egymás alá kerülnek (177 px szélesen, kilógás nélkül).
- `width:calc(230400px - 48000%)` a (480px − 100%) × 480 kifejtve, beágyazott zárójel nélkül. Ha a rács 480 px-nél keskenyebb, a tégla 100% széles lesz, egyébként a min-width érvényesül (177 px). A calc mindig az utolsó `width` legyen; ahol nem működik, ott kiesik, és az alap marad.

A 480 px alatti `@media` szabály maradjon (kártyás tégla), de így:
`.tile{display:block !important;width:100% !important;min-width:0 !important;max-width:100% !important;}`

**4.4. Tilos a `white-space:nowrap`** minden olyan elemen, amely 200 px-nél szélesebb lehet: gombokon, e-mail-címen, telefonszámon, linkeken, címkéken. Hosszú, szóköz nélküli szöveg (e-mail-cím, URL) tárolóján legyen `word-break:break-word;`.

**4.5. Képek:** minden tartalomkép `width="N"` attribútummal és `style="display:block;width:100%;max-width:Npx;height:auto;"` stílussal. A dekoratív sávok is így legyenek (borítókép, élek, hósáv, szalag). Fix pixeles szélesség csak 120 px alatti ikonon/logón lehet. **A képek méretén és arányán ne változtass**, csak a fenti stílus legyen mindenhol.

**4.6. Belső margók stílus nélkül:** a levél 320 px széles képernyőn, `<style>` nélkül se lógjon ki. A vízszintes belső margók (külső keret + szekció) összege legyen legfeljebb kb. 80 px, vagy a hosszú sorok törjenek.

**4.7. Csak világos megjelenés.** Sötét mód nincs:
- `<meta name="color-scheme" content="light only">`
- `<meta name="supported-color-schemes" content="light only">`
- a `<style>` elején: `:root{color-scheme:light only;supported-color-schemes:light only;}`
- **Törölni kell** minden `@media (prefers-color-scheme:dark)` szabályt, az Outlook.com sötétmód-szabályokat (`[data-ogsc]`, `[data-ogsb]`) és a csak ehhez tartozó `dm-*` osztályokat (v6 Tél, v9). A `-dark` változatú képek (pl. `tel-edge-dark.png`), ha csak a sötét módhoz kellettek, kikerülnek a sablonból.

**4.8. Kulcsok:** a `{{kulcs}}` készlet **pontosan ugyanaz maradjon**, mint most (99 kulcs + `assets.base`). Új kulcs nincs, egy sem tűnik el. Ellenőrizd gépileg, mint eddig.

## 5. A generátor ezekre a jelölésekre támaszkodik, ezeket meg kell őrizni

1. **Terméktéglák:** minden tégla `<div class="tile" …>` elemmel kezdődik, ahol a `class` az első attribútum. A téglák közvetlenül egymás után jönnek. Köztük pontosan ez a rejtett elválasztó áll:
   - `<!--[if mso]></td><td width="177" valign="top"><![endif]-->`
   - sorváltásnál: `<!--[if mso]></td></tr><tr><td width="177" valign="top"><![endif]-->`

   A 6 tégla tartalma betűre azonos, csak a sorszám más (`{{offer.items.1.name}}` … `{{offer.items.6.name}}`). Termékmező a rácson kívül nem szerepelhet. (A generátor ebből tetszőleges termékszámú rácsot készít.)
2. **Képviselő gombjai:** a telefon-gomb (`{{rep.phone.url}}`) és az e-mail-gomb (`{{rep.email.url}}`) **külön-külön** egy saját `<td>` cellában legyen, amely a gomb `<table>`-jét tartalmazza, a másik gombét nem. A hibrid oszlop-`div`-be is tehető így: `div > table > tr > td > table (gomb)`. (A generátor így rejti el a gombot, ha nincs adat.)
3. **Monogram:** `<td … width="N" …>{{rep.initials}}</td>`, a cellában csak a monogram (a generátor ide teszi a képviselő fotóját).
4. **Közösségi ikonok:** minden `{{social.…url}}` link saját `<td>` cellában legyen, egy cellában egy hálózat.
5. **Linkek:** `{{utility.browserLink.url}}` és `{{offer.more.url}}` egy-egy `<a href="{{…}}">` elem.

## 6. Átvételi próbák (mind a 4 sablonon, a mintatartalommal)

Bővítsd a `reszponziv-teszt.html`-t, hogy mind a 4 sablont megmutassa ezekben a nézetekben, és mindegyiken ellenőrizd:

| # | Nézet | Elvárás |
|---|---|---|
| A | 375 px széles keret, a sablon úgy, ahogy van | 1 oszlop, kártyás téglák (mint most), semmi nem lóg ki |
| B | 375 px és **320 px** széles keret, a `<style>` blokk **eltávolítva** | semmi nem lóg ki (a dokumentum szélessége = a keret szélessége), termékek egymás alatt |
| C | széles ablak (pl. 1280 px), a levél egy **360 px széles `div`**-ben, a `<style>` megmarad (így a média-lekérdezés nem lép életbe, mint Outlook webben) | semmi nem lóg ki, termékek egymás alatt |
| D | mint B és C, de minden `width:calc(…)` deklaráció törölve (Outlook) | semmi nem lóg ki, termékek egymás alatt (177 px) |
| E | 600 px és 800 px széles keret | pontosan a mostani asztali kinézet, 3 oszlopos rács |

„Semmi nem lóg ki” mérhetően: a keret `scrollWidth` értéke nem nagyobb a keret szélességénél. Írd ki a teszt-oldalra nézetenként a mért értéket (pl. „B 375 px: 375 / 375 ✓”).

## 7. Dokumentáció, hogy ez szabály maradjon

1. Új fájl a csomagban: **`RESZPONZIV-SZABALYOK.md`**: a fenti 2–6. pont a saját szavaiddal, kódmintákkal. Ez **projektszabály: minden jövőbeli sablon ezek szerint készül.**
2. A `MEZOK-ES-GENERATOR.md` elejére rövid szakasz „Reszponzivitás: kötelező felépítés” címmel, hivatkozással a fenti fájlra. Kerüljön bele a generátor által megőrzendő jelölések listája (5. pont) is.
3. `CHANGELOG.md`: új bejegyzés (1.13), amely sablononként felsorolja, mely sorok lettek hibridek, mi került ki (sötét mód), és hogy a kulcskészlet változatlan (gépileg ellenőrizve). Írd bele a 6. pont mért eredményeit is.
4. Ha van projektszintű utasításod vagy emlékeztetőd a sablonkészítéshez, abba is vedd fel: „Új sablon csak a RESZPONZIV-SZABALYOK.md szerint készülhet, és át kell mennie a reszponziv-teszt.html A–E próbáin.”

## 8. Amit kérek vissza

A 4 sablon a megszokott csomagszerkezetben, mindegyik a saját mappájában (`sablonok/`, `elonezet/`, `assets/`, mintatartalom-JSON, `CHANGELOG.md`, `MEZOK-ES-GENERATOR.md`, `RESZPONZIV-SZABALYOK.md`, `reszponziv-teszt.html`):

- `export-v1` (`v1-sotet-lemez`)
- `export-v2` (`v2-waterside`)
- `export-v6-tel` (`v6-tel`)
- `export-v9-kiarusitas` (`v9-kiarusitas`)

A kinézeten ne változtass. Ha valamelyik szabály egy sablonban látható változással járna, írd le a CHANGELOG-ban, mi és miért.
