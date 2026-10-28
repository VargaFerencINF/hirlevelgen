# Reszponzivitás: kötelező felépítés (projektszabály)

**Minden Energofish hírlevélsablon e szerint készül, a meglévők és a jövőbeliek is.** Új sablon csak akkor adható ki, ha átmegy a `reszponziv-teszt.html` A–E próbáin.

**A cél:** asztali gépen 3 oszlop, telefonon 1 oszlop. A levél semmilyen levelezőben nem lehet szélesebb a képernyőnél, akkor sem, ha a levelező az egész `<style>` blokkot eldobja.

---

## 1. Miért nem elég a média-lekérdezés

| Levelező | `<style>` / `@media` | `calc()` | Ami számít |
|---|---|---|---|
| Apple Mail (iPhone, Mac), Samsung Email | igen / igen | igen | minden réteg |
| Gmail app Google-fiókkal, Gmail web | igen / igen | igen | minden réteg |
| Gmail app **nem Google-fiókkal** (céges IMAP/Exchange) | nem / nem | nem | **csak az inline stílus** |
| Thunderbird | igen / nem | igen | inline stílus + calc |
| Outlook.com, új Outlook, Outlook web | igen / igen, de a **böngészőablakhoz** mér | nem | keskeny olvasópanelben a média-lekérdezés nem lép életbe |
| Outlook iOS / Android | részleges | nem | inline stílus, a média-lekérdezés bizonytalan |
| Klasszikus Outlook (Windows, Word-motor) | részleges | nem | rejtett `<!--[if mso]>` táblázatok, fix 600 px |
| Yahoo | igen / részleges | nem | inline stílus |

Ezért a reszponzivitás alapja az **inline stílus és a tartalomhoz igazodó (hibrid) felépítés**. A média-lekérdezés csak finomít; nélküle sem lóghat ki semmi.

## 2. A négy réteg

1. **Alapréteg (inline, egyedül is működik):** ami egymás mellett áll, az `display:inline-block` blokk, `width:100%` és fix pixeles **legnagyobb** szélesség (`max-width`). Ha nem fér el, magától új sorba kerül.
2. **calc-réteg (inline):** a terméktéglák a Fab Four technikával teljes szélességűek lesznek, ha a rács 480 px-nél keskenyebb. Ahol a `calc()` nem működik, az 1. réteg marad.
3. **Média-lekérdezés (`<style>`):** csak finomít: 640 px alatt a kétoszlopos sorok 50–50%-ban egymás mellett maradnak (mint a régi táblázatoknál), 480 px alatt a kártyás terméktégla, nagyobb betű, kisebb margó.
4. **Klasszikus Outlook:** minden többoszlopos sorhoz rejtett „ghost table” fix szélességekkel.

## 3. Kötelező szabályok

### 3.1. Külső keret
A külső táblázat `width="100%"`. A levéltest `width="600"` attribútummal és `style="width:100%;max-width:600px;"` stílussal, előtte a 600 px-es MSO-táblázat:

```html
<!--[if mso]><table role="presentation" cellpadding="0" cellspacing="0" border="0" width="600" align="center"><tr><td><![endif]-->
<table role="presentation" cellpadding="0" cellspacing="0" border="0" class="wrap" width="600" align="center" style="width:100%;max-width:600px;">
```

### 3.2. Hibrid oszlopok (nincs egymás melletti `<td>`)
Egymás melletti `<td>` cella nem lehet, ha a tartalmuk együtt 280 px-nél szélesebb lehet (fejléc felső sora, logó + dátum, ajánlat címke + jelvény, gombok, bélyegző, mezősor, lábléc). Helyette:

```html
<div style="font-size:0;line-height:0;text-align:left;">
<!--[if mso]><table role="presentation" cellpadding="0" cellspacing="0" border="0" width="536"><tr><td width="268" valign="top"><![endif]-->
<div class="col" style="display:inline-block;vertical-align:top;width:100%;max-width:268px;font-size:16px;line-height:24px;"><table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%"><tr><td>… bal oldal …</td></tr></table></div><!--[if mso]></td><td width="268" valign="top" align="right"><![endif]-->
<div class="col" style="display:inline-block;vertical-align:top;width:100%;max-width:268px;font-size:16px;line-height:24px;"><table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%"><tr><td class="col-r" align="right" style="text-align:right;">… jobb oldal …</td></tr></table></div>
<!--[if mso]></td></tr></table><![endif]-->
</div>
```

- A szülőn `font-size:0;line-height:0`, hogy a blokkok közé ne kerüljön hézag; a blokkon a betűméret és a sormagasság visszaáll.
- A `max-width` értékek összege = a tartalom szélessége (600 px − a szekció belső margói: 32 px-nél 536, 40 px-nél 520, a lap feletti sorban 584).
- A jobbra igazítást a blokkon belüli cella adja (`align="right"` + `text-align:right`); 480 px alatt a `.col-r` balra igazít.
- Ha a két oldal eltérő magas és a régi cellák középre igazítottak voltak (logó + dátum, lábléc), a blokk `vertical-align:middle`, az MSO-cella `valign="middle"`.
- Osztályok: `.col` (640 px alatt 50%, 480 px alatt 100%), `.col-h` (fejléc: 640 px alatt is 50%, telefonon egymás mellett marad), `.col-r` (480 px alatt balra).
- **Tartalomhoz méretezett gombok** (képviselő telefon- és e-mail-gombja): a blokk `display:inline-block;vertical-align:top;max-width:100%`, szélesség nélkül, így pontosan olyan széles, mint a gomb, és a két gomb asztalon ugyanúgy áll egymás mellett, mint eddig; ha nem fér, a második új sorba kerül. Az MSO-cellák itt automatikus szélességűek (a klasszikus Outlook így pontosan a régi elrendezést adja). 480 px alatt `.rep-col`: teljes szélességű gombok.
- **Kivétel, maradhat táblázatcella:** kis, fix szélességű kép vagy monogram (≤ 64 px) mellett tördelhető szöveg (aláíró, képviselő neve). A szövegcella összehúzódik, a sor nem lóghat ki, és a generátor a monogram-cellára támaszkodik.
- **Háromcellás sor** (v9 mezősor) és **szöveg + kép** sor (v9 bevezető + bélyegző): ugyanígy hibrid blokkok; 640 px alatt a `.trow` / `.tcell` osztály táblázatként jeleníti meg őket, hogy egyforma magasak maradjanak, mint eddig.

### 3.3. Termékrács (Fab Four)

```html
<div style="font-size:0;line-height:0;text-align:left;">
<!--[if mso]><table role="presentation" cellpadding="0" cellspacing="0" border="0" width="532"><tr><td width="177" valign="top"><![endif]-->
<div class="tile" style="display:inline-block;vertical-align:top;width:177px;min-width:177px;max-width:100%;width:calc(230400px - 48000%);">… {{offer.items.1.…}} …</div><!--[if mso]></td><td width="177" valign="top"><![endif]-->
<div class="tile" style="…ugyanaz…">… {{offer.items.2.…}} …</div><!--[if mso]></td><td width="177" valign="top"><![endif]-->
<div class="tile" style="…ugyanaz…">… {{offer.items.3.…}} …</div><!--[if mso]></td></tr><tr><td width="177" valign="top"><![endif]-->
<div class="tile" style="…ugyanaz…">… {{offer.items.4.…}} …</div><!--[if mso]></td><td width="177" valign="top"><![endif]-->
… 5., 6. …
<!--[if mso]></td></tr></table><![endif]-->
</div>
```

- Alap: `width:177px;min-width:177px;max-width:100%`. Asztalon 3 × 177 = 531 px egy sorban; keskeny helyen egymás alatt, 177 px szélesen.
- `width:calc(230400px - 48000%)` = (480 px − 100%) × 480, beágyazott zárójel nélkül. 480 px-nél keskenyebb rácsban a tégla 100% széles, egyébként a `min-width` (177 px) érvényes. **A calc mindig az utolsó `width`**; ahol nem működik, kiesik.
- 480 px alatt: `.tile{display:block !important;width:100% !important;min-width:0 !important;max-width:100% !important;}` és a meglévő kártyás tégla (kép balra 96 px, szöveg jobbra).
- Köztes sáv (média-lekérdezéssel, 481–590 px képernyő): ha a rács 480–530 px, két 177 px-es oszlop; 480 px alatti rácsban teljes szélességű téglák. 591 px felett 3 oszlop.

### 3.4. Nincs `white-space:nowrap`
Tilos minden olyan elemen, amely 200 px-nél szélesebb lehet: gomb, e-mail-cím, telefonszám, link, címke, mezőtartalom. Csak fix, rövid elemen maradhat (pl. a `&nbsp;&nbsp;·` elválasztó). Szóköz nélküli hosszú szöveg (e-mail-cím, URL) tárolóján `word-break:break-word;`. A nagy főcímen is `word-break:break-word;`, hogy egy hosszú összetett szó se tolja szét a levelet.

### 3.5. Képek
Minden tartalomkép és dekoratív sáv (borító, hullám, él, hósáv, szalag, termékkép):

```html
<img src="…" width="600" style="display:block;width:100%;max-width:600px;height:auto;">
```

Fix pixeles szélesség csak 120 px alatti ikonon és logón lehet. A képek mérete és aránya nem változik.

### 3.6. Belső margók stílus nélkül
320 px-es képernyőn, `<style>` nélkül sem lóghat ki semmi. A vízszintes margók összege (külső keret + szekció) legfeljebb kb. 80 px, különben a hosszú soroknak törniük kell. A külső cella inline vízszintes margója ezért **6 px** (ugyanannyi, mint amit eddig a 480 px alatti média-lekérdezés adott); asztalon ez nem látszik, mert a 600 px-es levél középen áll, és 1024 px alatt a média-lekérdezés felülírja.

### 3.7. Csak világos megjelenés

```html
<meta name="color-scheme" content="light only">
<meta name="supported-color-schemes" content="light only">
<style>
:root{color-scheme:light only;supported-color-schemes:light only;}
```

Nincs `@media (prefers-color-scheme:dark)`, nincs `[data-ogsc]` / `[data-ogsb]`, nincs `dm-*` osztály, nincs csak sötét módhoz tartozó kép.

### 3.8. Kulcsok
A `{{kulcs}}` készlet pontosan ugyanaz marad: 99 kulcs + `assets.base`, ugyanabban a sorrendben. Új kulcs nincs, egy sem tűnik el. Minden módosítás után gépileg ellenőrizni.

## 4. A generátor ezekre a jelölésekre támaszkodik (megőrzendő)

1. **Terméktéglák:** minden tégla `<div class="tile" …>`, a `class` az első attribútum. A téglák közvetlenül egymás után jönnek, köztük pontosan ez áll:
   - `<!--[if mso]></td><td width="177" valign="top"><![endif]-->`
   - sorváltásnál: `<!--[if mso]></td></tr><tr><td width="177" valign="top"><![endif]-->`

   A 6 tégla tartalma betűre azonos, csak a sorszám más (`{{offer.items.1.name}}` … `{{offer.items.6.name}}`). Termékmező a rácson kívül nem szerepelhet.
2. **Képviselő gombjai:** a telefon-gomb (`{{rep.phone.url}}`) és az e-mail-gomb (`{{rep.email.url}}`) külön-külön egy saját `<td class="rep-btn">` cellában, amely csak a saját gombja `<table>`-jét tartalmazza: `div.rep-col > table > tr > td.rep-btn > table (gomb)`.
3. **Monogram:** `<td … width="64" …>{{rep.initials}}</td>`, a cellában csak a monogram.
4. **Közösségi ikonok:** minden `{{social.…url}}` link saját `<td>` cellában, cellánként egy hálózat.
5. **Linkek:** `{{utility.browserLink.url}}` és `{{offer.more.url}}` egy-egy `<a href="{{…}}">` elem.

## 5. Átvételi próbák (`reszponziv-teszt.html`)

Minden sablon (jelenleg 5: v1, v2, v6 Ősz, v6 Tél, v9 Kiárusítás), a mintatartalommal. „Semmi nem lóg ki” = a keret `scrollWidth` értéke nem nagyobb a keret szélességénél; az oldal nézetenként kiírja (pl. „B 375 px: 375 / 375 ✓”).

| # | Nézet | Elvárás |
|---|---|---|
| A | 375 px, a sablon változatlanul | 1 oszlop, kártyás téglák, semmi nem lóg ki |
| B | 375 és 320 px, `<style>` nélkül | semmi nem lóg ki, termékek egymás alatt |
| C | 1280 px ablak, a levél 360 px-es div-ben, `<style>` marad | semmi nem lóg ki, termékek egymás alatt |
| D | mint B és C, minden `width:calc(…)` törölve | semmi nem lóg ki, termékek egymás alatt, 177 px |
| E | 600 és 800 px | a megszokott asztali kinézet, 3 oszlopos rács |

A próbaoldalt webszerverről kell megnyitni (pl. `npx serve` a csomagok szülőmappájában), és az öt csomagmappa (export-v1, export-v2, export-v6, export-v6-tel, export-v9-kiarusitas) legyen egymás mellett.

## 6. Ellenőrzőlista új sablonhoz

- [ ] Külső keret a 3.1 szerint.
- [ ] Nincs egymás melletti `<td>` (a 3.2 kivételén túl); minden többoszlopos sor hibrid, ghost table-lel.
- [ ] Termékrács a 3.3 mintája szerint, a generátor jelöléseivel (4. pont).
- [ ] Nincs `white-space:nowrap` 200 px-nél szélesebb elemen; e-mail-cím és főcím törhető.
- [ ] Minden kép a 3.5 szerint.
- [ ] 320 px, `<style>` nélkül nem lóg ki semmi.
- [ ] Csak világos megjelenés.
- [ ] Kulcskészlet gépileg ellenőrizve.
- [ ] `reszponziv-teszt.html`: mind a 9 nézet ✓.
