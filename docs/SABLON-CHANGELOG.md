# Changelog: Energofish hírlevél-sablonok

## 1.14 · 2026-10-28 · Export a ZIP-szabályok szerint (EXPORT-SZABALYOK.md)

Ettől a változattól minden export az `EXPORT-SZABALYOK.md` szerint készül: sablononként egy ZIP (`Energofish_Partner_Newsletter_Design_v9-kiarusitas.zip`), benne egyetlen gyökérmappa (`export-v9-kiarusitas/`). A sablonfájl neve az azonosító: `sablonok/v9-kiarusitas.html`, **az előző változattal azonos**, így az alkalmazás a meglévő sablont frissíti, nem hoz létre újat.

**Mezők és hivatkozások: nincs változás.** A kulcskészlet (99 kulcs + `assets.base`) és a kulcsok sorrendje gépileg ellenőrizve. A generátoron nem kell módosítani.

Ebben a csomagban:
- Új fájl: `EXPORT-SZABALYOK.md` (a ZIP-szerkezet, az elnevezés, a képek és a méretkorlátok szabályai, ellenőrzőlistával). Minden csomagban ugyanez a fájl.
- `MEZOK-ES-GENERATOR.md`: a csomagszerkezet az új szabály szerint (ZIP-név, gyökérmappa, a képek pontos listája), hivatkozással az `EXPORT-SZABALYOK.md`-re.
- **Logó:** az `energofish-mark.png` ebben a csomagban bájtra eltért a v2 azonos nevű fájljától, a képpontjai viszont azonosak voltak (csak a PNG-tömörítés különbözött). A 4. pont szerint („egy fájlnév = egy kép”) most a közös fájl szerepel; a kép tartalma nem változott.

**Ellenőrzőlista** (gépileg, a kész ZIP visszaolvasott tartalmán):
- [x] A ZIP-ben egy gyökérmappa van: `export-v9-kiarusitas/`.
- [x] A `sablonok/` mappában pontosan egy HTML van: `v9-kiarusitas.html`, azonos az előző változat nevével.
- [x] Csak a sablonfájlban van kapcsos zárójelpár; az `elonezet/v9-kiarusitas.html` és a `reszponziv-teszt.html` fájlban nincs.
- [x] Mind a 7 hivatkozott kép (`{{assets.base}}/…`) megvan az `assets/`-ben, egyező névvel, PNG vagy JPG; külső képhivatkozás nincs. Az `assets/` 9 fájlja: 7 hivatkozott sablonkép + 2 mintatartalom-kép, más nincs.
- [x] Nincs megváltozott tartalmú kép régi néven: minden fájlnév minden csomagban bájtra azonos képet jelent (SHA-256-tal összevetve az 5 csomag 49 képén).
- [x] A sablon 40.9 KB (≤ 90 KB), a `<style>` 3.6 KB (≤ 16 KB); a legnagyobb kép 115 KB (`cover-kiarusitas-1200x660.jpg`, ≤ 1 MB).
- [x] A kulcskészlet változatlan (gépileg ellenőrizve).
- [x] A `reszponziv-teszt.html` A–E próbái mind sikeresek: mind az 5 sablon mind a 9 nézetben ✓ (45/45, 2026-10-28).
- [x] Nincs rejtett vagy rendszerfájl, beágyazott ZIP vagy szimbolikus link a ZIP-ben; Deflate tömörítés, UTF-8 fájlnevek (UTF-8 jelzővel), UTF-8 kódolású HTML/JSON/MD.

## 1.13 · 2026-10-28 · Reszponzivitás stílus nélkül is (hibrid felépítés), csak világos megjelenés

**Mezők és hivatkozások: nincs változás.** Gépileg ellenőrizve mind a négy sablonon: a kulcskészlet azonos (99 kulcs + `assets.base`), a kulcsok sorrendje is azonos. A `{{assets.base}}` előfordulásainak száma a v6 Télben 9 → 7, a v9 Kiárusításban 11 → 9, mert kikerült a két-két, csak sötét módhoz tartozó kép. A 6 terméktégla tartalma sablononként betűre azonos (csak a sorszám más), termékmező a rácson kívül nincs, a tégla-elválasztók változatlanok. A generátoron nem kell módosítani.

Érintett: `sablonok/` és `elonezet/`: `v1-sotet-lemez`, `v2-waterside`, `v6-tel`, `v9-kiarusitas` (mindegyik a saját csomagjában).

Új fájlok minden csomagban: `RESZPONZIV-SZABALYOK.md` (projektszabály, kódmintákkal), `reszponziv-teszt.html` (A–E próbák, mért értékekkel). A `MEZOK-ES-GENERATOR.md` elején új szakasz: „Reszponzivitás: kötelező felépítés”, a generátor által megőrzendő jelölésekkel.

### Minden sablonban
- **Terméktégla (Fab Four):** `width:177px;min-width:177px;max-width:100%;width:calc(230400px - 48000%)`. A 480 px alatti szabály: `.tile{display:block;width:100%;min-width:0;max-width:100%}` (a kártyás tégla megmaradt).
- **Hibrid oszlopok** (`inline-block` + `max-width`, rejtett MSO-táblázattal) a korábbi egymás melletti cellák helyett, sablononként lásd lent.
- **Képviselő gombjai:** két tartalomhoz méretezett hibrid blokk (`.rep-col`), mindkét gomb saját `<td class="rep-btn">` cellában. Asztalon ugyanúgy állnak egymás mellett, mint eddig; ha nem férnek, a második új sorba kerül.
- **`white-space:nowrap` kikerült:** a gombokról (telefon, e-mail) és a borító meta-sorának mezőiről (szám, olvasási idő, termékszám). Csak a rövid `&nbsp;&nbsp;·` elválasztón maradt. Az e-mail- és a telefon-gombon `word-break:break-word`, a főcímen is.
- **Külső cella inline vízszintes margója 12 → 6 px** (a 4.6 szabály miatt).
- **640 px alatti média-lekérdezés:** a kétoszlopos sorok 50–50%-ban egymás mellett maradnak, ahogy eddig a táblázatcellák. **480 px alatt** a `.col` sorok egymás alá kerülnek (balra igazítva), mint eddig a `.stack` cellák; a fejléc logója és dátuma telefonon is egymás mellett marad (`.col-h`).
- **Sötét mód kikerült:** `color-scheme` és `supported-color-schemes`: `light only` (meta és `:root`). A `.stack` és a `.rep-btn a` szabály megszűnt (nincs rá szükség).

### Sablononként: hibrid sorok és törölt elemek
- **v1 Sötét lemez** (tartalom 536 px): fejléc felső sora (268 + 268), logó + dátum (268 + 268, középre igazítva), ajánlat címke + jelvény (268 + 268), képviselő gombjai.
- **v2 Waterside** (536 px): ugyanazok a sorok, mint a v1-ben.
- **v6 Tél** (520 px): fejléc felső sora a lap felett (292 + 292), logó + dátum a sötét lemezen (260 + 260), ajánlat címke + jelvény (260 + 260), képviselő gombjai, lábléc logó + közösségi ikonok (260 + 260).
  - Sötét mód törölve: `@media (prefers-color-scheme:dark)`, `[data-ogsc]`/`[data-ogsb]` szabályok, minden `dm-*` osztály, a sötét változatú arany él (`tel-edge-dark.png`) és a lábléc sötét módú logója. A `tel-edge-dark.png` a csomag `assets/` mappájából is kikerült.
- **v9 Kiárusítás** (520 px): fejléc felső sora (292 + 292), logó + dátum (260 + 260), bevezető + bélyegző (412 + 108; 480 px alatt a bélyegző rejtve, mint eddig), háromcellás mezősor `.mcell` (156 + 208 + 156), ajánlat címke + arany jelvény (260 + 260; a jelvényről kikerült a `nowrap`), képviselő gombjai, lábléc logó + közösségi ikonok (260 + 260).
  - A mezősor és a bevezető sora 640 px alatt táblázatként jelenik meg (`.trow`/`.tcell`), így a cellák egyforma magasak maradnak, és a mezősor telefonon is egy sorban marad, mint eddig.
  - Sötét mód törölve: `@media (prefers-color-scheme:dark)`, `[data-ogsc]`/`[data-ogsb]`, minden `dm-*` osztály, a fejléc és a lábléc sötét módú logója. Az `energofish-mark-light.png` ebben a csomagban már nem kell, kikerült az `assets/` mappából.

### Látható változások és okuk
Asztalon (600 px és szélesebb) és a klasszikus Outlookban a kinézet változatlan (E próba). Ami máshol változik:
- **Média-lekérdezés nélkül** (Gmail app nem Google-fiókkal, keskeny Outlook-olvasópanel): a kétoszlopos sorok egymás alá kerülnek, a termékek egymás alatt, teljes szélességben (calc nélkül 177 px szélesen). Eddig ezek a sorok és a 3 oszlopos rács kilógtak vagy összenyomódtak. Ok: 4.2, 4.3.
- **481–590 px széles képernyőn** (média-lekérdezéssel): a rács 540–590 px között 2 oszlopos (177 px-es téglák, balra igazítva), 481–539 px között teljes szélességű téglák. Eddig 3 összenyomott oszlop volt. Ok: a kötelező 177 px-es legkisebb téglaszélesség és a 480 px-es calc-küszöb (4.3).
- **Főcím, e-mail-cím:** ha egy szó nem fér el a sorban, szón belül törik (eddig kilógott). Asztalon nem fordul elő. Ok: 4.4, 4.6.
- **Borító meta-sora:** keskeny képernyőn egy mezőn belül is törhet (eddig csak a mezők között). Ok: 4.4.
- **Külső margó 6 px** stílus nélküli, keskeny nézetben (eddig 12 px). Ok: 4.6.
- **Sötét módú levelezőben** a levél világos marad (a v6 Tél és a v9 Kiárusítás eddig saját sötét színeket kapott). Egyes alkalmazások (pl. Gmail, Outlook app) a `light only` jelzés ellenére is részben átszínezhetik; ez a levelezőn múlik. Ok: 4.7.

### Mért eredmények (`reszponziv-teszt.html`, mintatartalommal, 2026-10-28)
Mind a 36 nézet ✓ (keret `scrollWidth` / keretszélesség).

| Nézet | v1 Sötét lemez | v2 Waterside | v6 Tél | v9 Kiárusítás |
|---|---|---|---|---|
| A 375 px | 375 / 375 ✓ kártyás, egymás alatt | 375 / 375 ✓ kártyás, egymás alatt | 375 / 375 ✓ kártyás, egymás alatt | 375 / 375 ✓ kártyás, egymás alatt |
| B 375 px, stílus nélkül | 375 / 375 ✓ egymás alatt | 375 / 375 ✓ egymás alatt | 375 / 375 ✓ egymás alatt | 375 / 375 ✓ egymás alatt |
| B 320 px, stílus nélkül | 320 / 320 ✓ egymás alatt | 320 / 320 ✓ egymás alatt | 320 / 320 ✓ egymás alatt | 320 / 320 ✓ egymás alatt |
| C 360 px div, 1280 px ablak | 360 / 360 ✓ egymás alatt | 360 / 360 ✓ egymás alatt | 360 / 360 ✓ egymás alatt | 360 / 360 ✓ egymás alatt |
| D 375 px, stílus és calc nélkül | 375 / 375 ✓ 177 px | 375 / 375 ✓ 177 px | 375 / 375 ✓ 177 px | 375 / 375 ✓ 177 px |
| D 320 px, stílus és calc nélkül | 320 / 320 ✓ 177 px | 320 / 320 ✓ 177 px | 320 / 320 ✓ 177 px | 320 / 320 ✓ 177 px |
| D 360 px div, calc nélkül | 360 / 360 ✓ 177 px | 360 / 360 ✓ 177 px | 360 / 360 ✓ 177 px | 360 / 360 ✓ 177 px |
| E 600 px | 600 / 600 ✓ 3 oszlop | 600 / 600 ✓ 3 oszlop | 600 / 600 ✓ 3 oszlop | 600 / 600 ✓ 3 oszlop |
| E 800 px | 800 / 800 ✓ 3 oszlop | 800 / 800 ✓ 3 oszlop | 800 / 800 ✓ 3 oszlop | 800 / 800 ✓ 3 oszlop |

> **Kiegészítés (2026-10-28):** a v6 Ősz (`export-v6`, `v6-osz-avar`) ugyanígy átalakítva. A `reszponziv-teszt.html` most mind az 5 sablont mutatja; mind a 45 nézet ✓.

## 1.12 · 2026-10-27 · Új sablon: v9 Kiárusítás („A raktár kiürül”)

**Mezők és hivatkozások: nincs változás.** A v9 Kiárusítás kulcskészlete a v1-ével azonos, a sorrend is (gépileg ellenőrizve, 99 kulcs + `assets.base`). **Nincs kérdés-modul** (`poll.*`, `cover.meta.poll`): a v4/v5 kérdéses kulcskészlete helyett a projektszabály szerint a v1/v2 kulcsai. Táblázatszerkezet, osztályok, töréspontok és termékrács a meglévő sablonokéval azonos (3 oszlop; 480 px alatt az 1.3 óta érvényes egymás alatti tégla). Mobilon négy apró kiegészítő szabály van (`.hide-m`, `.chip`, `.mcell`, `.mcell1`), a terméktégla `t-code` osztálya `t-tag` lett. A generátorban csak a sablonlistába kell felvenni a `v9-kiarusitas`-t. A korábbi sablonok nem változtak.

> **Elnevezés:** a „v9” sorszámot az 1.7-es `v9-black-friday-grafit` is használja; a mappa- és fájlnevek különböznek (`export-v9-kiarusitas/`, `v9-kiarusitas.html`).

Új fájlok: `sablonok/v9-kiarusitas.html`, `elonezet/v9-kiarusitas.html`, `tartalom-v9-kiarusitas.json`, `assets/kiar-tape.png`, `assets/kiar-stamp.png`, `assets/cover-kiarusitas-1200x660.jpg` (Unsplash-stockfotó, mezőtartalom).

Koncepció: **raktári papírmunka.** Kiürülő raktár, pontosan dokumentálva: a levél egy szállítólevél, a téglák készletkártyák. Világos felületek, a sürgetést a határidő, a darabszám és a rendezett szerkezet adja, nem a kiabálás.
- **Keret:** beton-szürke lap (#E6E8EA), fehér papírlap 10 px-es sarokkal, tetején és alján vékony arany-grafit figyelmeztető szalag.
- **Fejléc:** világos dokumentumfej, sötét logóval.
- **Borító:** a fotó a lap margóján belül, mint egy csatolt kép. Alatta szállítólevél-blokk: arany címke a zárással (`cover.kicker`), főcím, bevezető mellette szöveg nélküli bélyegzővel, és vonalazott mezősor (szám, olvasási idő, tételszám) táblázatos számjegyekkel.
- **Elválasztó:** szaggatott letépési vonal.
- **Ajánlat:** készletlap-fejléc két vastag vonal között, jobbra arany címkén a tételszám és a zárás.
- **Készletkártya:** kép, alatta kartonszínű polccímke a cikkszámmal (szaggatott perforációval), név, leírás, 22 px-es ár, alatta arany leértékelő címke a kedvezménnyel és a maradék darabszámmal.
- **Képviselő:** letéphető kartonszelvény (szaggatott felső él, #F2E8D8), a nagyobb tételek telefonos foglalására.
- **Lábléc:** a szalag zárja a lapot, alatta a megszokott lábléc a beton-szürke lapon.
- **Szabályok:** minden szöveg élő és egyenes; a mintázat (szalag) csak szöveg nélküli sávban van; arany felületen mindig #1A171E szöveg (8,8:1); kikapcsolt képeknél a háttérszínek viszik a szerkezetet.
- **Mintatartalom:** „Utolsó dobás: ami a raktárban maradt”, „Kifogyóban”, hat tétel 20–35% kedvezménnyel és maradék darabszámmal, zárás 2026. november 13., péntek 12:00.

## 1.11 · 2026-10-26 · v6 Tél: finomított téli megjelenés

**Mezők és hivatkozások: nincs változás** (a v1 kulcskészlete, gépileg ellenőrizve). Csak sablonképek és két sorközi ornamens változott.
- **Hóesés fotószerűen:** a clipart-szerű hópehely-ikonok helyett lágy, kerek pelyhek három mélységben (apró távoli, közepes, néhány életlen előtérbeli). A minta-borítókép teteje kissé sötétebb lett, így a pelyhek finoman kirajzolódnak.
- **Hósáv** (`tel-snow-band.png`, 600×36): a borító hóesése folytatódik a sötét lemezen, lefelé elhalványul. Ez a sablon része, cserélt borítóképnél is megmarad.
- **Arany él:** a kristályok kikerültek, a borítósáv alján tiszta, 6 px-es arany él van, mint a v1-ben (`tel-edge-light/dark.png`, 600×14).
- **Ornamens:** egy pontos, vékony vonalas jégkristály a borító felülcíme, az ajánlat és a képviselő címkéje előtt (`tel-marker.png`, új: `tel-marker-gold.png`). Ez az egyetlen ünnepi utalás.

## 1.10 · 2026-10-26 · Törölve: v4 Partnerjelentés és v5 Ősz

A két sablon kérdés-modult tartalmazott („Egy kérdés”, `poll.*`, `cover.meta.poll`), ezért mindkettő törölve lett. Kérdés-modul ezentúl egyik sablonba sem kerül, minden sablon a v1/v2 kulcskészletét használja.
- Törölve: `sablonok/` és `elonezet/` `v4-partnerjelentes.html`, `v5-osz.html`; `tartalom-v5-osz.json`; a csak hozzájuk tartozó képek (`contour-band.png`, `leaf-*.png`, `cover-osz-helyorzo.jpg`).
- **A generátorban:** a `v4-partnerjelentes` és a `v5-osz` kerüljön ki a sablonlistából. A `poll.*` és a `cover.meta.poll` mező a tartalom-JSON-ból elhagyható.
- A `MEZOK-ES-GENERATOR.md` kulcstáblázataiból kikerült a v4 oszlop és a kérdés-mezők. A `reszponziv-teszt.html` már csak a v1-et és a v2-t mutatja.

## 1.9 · 2026-10-26 · Új sablon: v6 Tél

**Mezők és hivatkozások: nincs változás.** A v6 Tél kulcskészlete a v1-ével (és a v2-ével) azonos, a kulcsok sorrendje is (gépileg ellenőrizve: 99 kulcs, egy sem új, egy sem hiányzik). **Nincs benne kérdés-modul:** nincs `poll.*` és nincs `cover.meta.poll`. A táblázatszerkezet, az osztályok és a töréspontok a meglévő sablonokéval azonosak. A generátorban csak a sablonlistába kell felvenni a `v6-tel`-t. A korábbi sablonok nem változtak.

> **Elnevezés:** a „v6” sorszámot az 1.4-es `v6-osz-avar` sablon is használja. A fájlnevek különböznek (`v6-osz-avar.html`, `v6-tel.html`), így nem ütköznek.

Új fájlok:
- `sablonok/v6-tel.html`, `elonezet/v6-tel.html`, `tartalom-v6-tel.json` (téli mintatartalom, a v1 kulcskészletével, kérdés nélkül);
- `assets/tel-edge-light.png`, `assets/tel-edge-dark.png`, `assets/tel-marker.png`;
- `assets/tel-snow-band.png`;
- `assets/cover-tel-1200x660.jpg`: minta-borítókép (mezőtartalom, cserélhető).

Koncepció: **„Tél a víznél”.** Csendes, deres tópart hajnalban: köd a nyílt víz felett, dér a nád hegyén, és a felkelő nap az egyetlen meleg fény. A hideg felületek között a márka aranya a fény és a melegség.
- **Színek:**
  - a lap háttere a rendszer Víz tónusa (#DCE8F2);
  - a sötét lemez téli mélyvíz-kék (#16232C): fejléc, a borító szövege, képviselő-blokk;
  - az elválasztó vonal és a terméktéglák kerete hideg tónusú (#C9D7E3, #C4D2DE);
  - az arany él, az arany monogramkör és az arany telefon-gomb változatlan.
- **Jégkristályok (2 helyen):**
  - a borítósáv és a levél határán, a borító szövege alatti sávban: két jégkék és egy arany kristály (egy kép, sötét és világos változattal);
  - egy kis arany kristály az ajánlat címkéje előtt.
  - Az arany kristály a visszafogott ünnepi utalás. Nincs Mikulás, rénszarvas vagy karácsonyi clipart.
- **Hópelyhek, statikusan:** egy hópelyhes sáv (`tel-snow-band.png`) a borítófotó és a borító szövege között, ugyanolyan `<tr><td><img></td></tr>` sor, mint a borítókép, így minden kliensben látszik. A minta-borítóképbe is bele vannak dolgozva a pelyhek. Nincs animáció.
- **Sötét mód:** `prefers-color-scheme` és Outlook.com szabályok, hideg tónusokkal (lap #0F181F, levél #26333D, vonal #3A4955).
- **Mobil:** a 480 px alatti terméktéglák belső elválasztó vonala hideg tónusú (#E1E9EF). Ez egy értékmódosítás a meglévő szabályban, új szabály nincs.
- **Mintatartalom:** év végi köszönet, előrendelés a 2027-es szezonra (január 31-ig −8%) és hat ajándéknak is jó termék az ünnepi polcra (december 20-ig).

## 1.8 · 2026-10-21 · Új sablon: v10 Black Friday (éjszaka)

**Mezők és hivatkozások: nincs változás.** A v10 is a működő `v1-sotet-lemez.html` kódjára épül, ugyanaz a táblázatszerkezet, ugyanazok az osztályok és töréspontok. A kulcskészlet a v1-ével azonos (gépileg ellenőrizve). A generátorban csak a sablonlistába kell felvenni a `v10-black-friday-ejszaka`-t.

Új fájlok:
- `sablonok/v10-black-friday-ejszaka.html`, `elonezet/v10-black-friday-ejszaka.html`, `tartalom-v10-black-friday-ejszaka.json`;
- `assets/cover-black-friday-ejszaka-1200x660.jpg`: minta-borítókép (mezőtartalom, cserélhető).

Koncepció: **„Black Friday éjszaka”.** Éjszakai horgászat a Balatonon: a kedvezmény a telihold, a fénye aranycsíkként tükröződik a vízen.
- **Színek:** éjkék (#10263A, a Víz kék sötétített árnyalata) és mélyebb éjkék (#0D1F2F). Nem fekete és nem szürke. Arany kiemelés (#F1A32B), meleg Hajnal krém (#FCE6C8) képviselő-blokk.
- **Borító:** éjkék duotónus fotó, arany hold „akár −55%” felirattal, a hold tükröződése a vízen, alatta négy naptárlap (27–30., az első arany).
- **Borító szövege:** a három meta-adat (szám, olvasási idő, tételszám) kapszulaként jelenik meg, az első arany. Csak a megjelenés változott, a mezők ugyanazok.
- **Ajánlat:** fehér kártyák éjkék alapon, arany „Rendelés” gomb, aranykapszula a tételszámnál. A „Teljes kínálat” link arany keretes gomb lett (ugyanaz a `offer.more` mező).
- **Arany él** a sötét felületek alján (a márka „sötét lemez, arany él” motívuma).
- **Teljesen statikus:** nincs háttérkép, nincs animáció, nincs sötétmód-szabály.

## 1.7 · 2026-10-20 · Új sablon: v9 Black Friday (grafit)

**Mezők és hivatkozások: nincs változás.** A v9 is a működő `v1-sotet-lemez.html` kódjára épül, ugyanaz a táblázatszerkezet, ugyanazok az osztályok és töréspontok. A kulcskészlet a v1-ével azonos (gépileg ellenőrizve). A v7 (sötét) és a v8 (világos) megmarad, a v9 egy harmadik irány ugyanarra a kampányra. A generátorban csak a sablonlistába kell felvenni a `v9-black-friday-grafit`-ot.

Új fájlok:
- `sablonok/v9-black-friday-grafit.html`, `elonezet/v9-black-friday-grafit.html`, `tartalom-v9-black-friday-grafit.json`;
- `assets/bf-perf-a.png`, `bf-perf-b.png`, `bf-perf-c.png`: perforált, „letéphető kupon” szegélyek a színváltásoknál (1200×40, 600 px széles);
- `assets/cover-black-friday-grafit-1200x660.jpg`: minta-borítókép (mezőtartalom, cserélhető).

Eltérések a v1-től:
- **Grafit alap, nem fekete:** márkaszürke (#414242) fejléc, borítószöveg és ajánlatsáv, mélyebb grafit (#2E2F31) felső sáv és lábléc, meleg törtfehér (#F4F1EC) képviselő-blokk.
- **Árcédula-koncepció:** a borítón duotónus fotó, rajta arany árcédula („akár −55%”) zsinórral. A színváltásoknál perforált szegélyek vannak, így a levél egy letéphető kuponra emlékeztet.
- **Termékkártyák:** grafit kártya fehér képmezővel, fehér név és ár, arany akciósor, arany „Rendelés” gomb.
- **Mobil:** a kártyákon belüli elválasztó vonal grafitszínű (#5C5D60), a kép bal felső sarka lekerekített. Ez két CSS-értékmódosítás a meglévő 480 px-es szabályokban, új szabály nincs.
- **Teljesen statikus:** nincs háttérkép, nincs animáció, nincs sötétmód-szabály.

## 1.6 · 2026-10-20 · Új sablon: v8 Black Friday (világos)

**Mezők és hivatkozások: nincs változás.** A v8 is a működő `v1-sotet-lemez.html` kódjára épül, ugyanaz a táblázatszerkezet, osztályok és töréspontok. A kulcskészlet a v1-ével azonos (gépileg ellenőrizve). A v7 (sötét) megmarad, a v8 egy világos alternatíva ugyanarra a kampányra. A generátorban csak a sablonlistába kell felvenni a `v8-black-friday-light`-ot.

Új fájlok:
- `sablonok/v8-black-friday-light.html`, `elonezet/v8-black-friday-light.html`, `tartalom-v8-black-friday-light.json`;
- `assets/energofish-mark.png`: sötét logó a fehér fejléchez;
- `assets/cover-black-friday-light-1200x660.jpg`: minta-borítókép (mezőtartalom, cserélhető).

Eltérések a v1-től:
- **Világos alap:** fehér fejléc sötét logóval, alatta 2 px-es fekete vonal. Fehér borítószöveg, világosszürke ajánlatsáv, fehér képviselő-blokk. Egyetlen sötét felület a lábléc, alatta az arany sáv.
- **Borító:** osztott minta-kép (bal oldalon tipográfia, jobb oldalon a Balaton-fotó, a kettő határán fekete „akár −55%” kör). Alatta 40 px-es cím.
- **Termékkártyák:** lekerekített fehér kártya, finom keret, fekete „Rendelés” gomb.
- **Képviselő:** fekete telefon-gomb, körvonalas e-mail gomb, arany monogramkör.
- **Teljesen statikus:** nincs háttérkép, nincs futószalag, nincs animáció.

## 1.5 · 2026-10-20 · Új sablon: v7 Black Friday

**Mezők és hivatkozások: nincs változás.** A v7 is a működő `v1-sotet-lemez.html` kódjára épül: ugyanaz a táblázatszerkezet, ugyanazok az osztályok és töréspontok. A kulcskészlet a v1-ével azonos (gépileg ellenőrizve). A generátorban csak a sablonlistába kell felvenni a `v7-black-friday`-t.

Új fájlok:
- `sablonok/v7-black-friday.html`, `elonezet/v7-black-friday.html`, `tartalom-v7-black-friday.json`;
- `assets/bf-ticker.png`: „BLACK FRIDAY” futószalag (1200×96, 600 px széles), a borító alatt és a lábléc végén;
- `assets/cover-black-friday-1200x660.jpg`: minta-borítókép (mezőtartalom, cserélhető);
- `assets/social-*.png`: eredeti színű közösségi ikonok (mint a v6-ban).

Eltérések a v1-től:
- **Színek:** mélyfekete lemez (#0E0D10), sötét keret (#1A171E), arany kiemelés (#F1A32B). A ritmus: sötét borító, fehér levél, sötét ajánlat, fehér képviselő, sötét lábléc.
- **Borító:** 46 px-es cím fehéren, arany felülcím és pontok.
- **Termékkártyák:** fehér kártya fekete lemezen, 4 px-es arany felső él, fekete „Rendelés” gomb.
- **Képviselő:** fehér blokk, fekete monogramkör arany betűvel.
- **Teljesen statikus:** nincs háttérkép, nincs animáció, nincs sötétmód-szabály.

## 1.4 · 2026-10-20 · Új sablon: v6 Ősz (avar)

**Mezők és hivatkozások: nincs változás.** A v6 a működő `v1-sotet-lemez.html` kódjára épül, sorról sorra ugyanaz a táblázatszerkezet és ugyanaz a mobil CSS. Csak a színek változtak, és néhány statikus kép került bele. A kulcskészlet a v1-ével azonos (gépileg ellenőrizve). A generátorban csak a sablonlistába kell felvenni a `v6-osz-avar`-t.

Új fájlok:
- `sablonok/v6-osz-avar.html`, `elonezet/v6-osz-avar.html`, `tartalom-v6-osz.json`;
- `assets/osz-edge.png`, `osz-footer.png`, `osz-mark-maple.png`, `osz-mark-oak.png`, `osz-mark-birch.png`;
- `assets/social-facebook.png`, `social-youtube.png`, `social-instagram.png`, `social-tiktok.png`: eredeti színű közösségi ikonok, 72×72 px, 36 px-en jelennek meg.

Eltérések a v1-től:
- **Színek:** sötétbarna lemez (#2B1E16), krém lap (#FFFBF4), Homok ajánlatsáv (#F2E8D8), rozsda képviselő-blokk (#8C3B1C), arany „Rendelés” gombok.
- **Borító felirata** sötét lemezen. Alatta egy új képsor (`osz-edge.png`), a lábléc után egy másik (`osz-footer.png`). Mindkettő ugyanolyan `<tr><td><img></td></tr>` sor, mint a borítókép.
- **Kis levél ikon** a borító felirata, az ajánlat és a képviselő címkéje előtt (sorközi kép).
- **Közösségi ikonok:** az egyszínű `images.energofish.hu` ikonok helyett eredeti színű ikonok.
- **Nincs háttérkép**, nincs animáció, nincs sötétmód-szabály: teljesen statikus.

## 1.3 · 2026-10-19 · Mobilon egymás alatt a termékek

**Mezők és hivatkozások: nincs változás.** Egyetlen `{{kulcs}}` sem változott, a generátoron nem kell módosítani.
- **Termékrács mobilon (max. 480 px):** a 2 oszlopos rács helyett a termékek egymás alatt, teljes szélességben jelennek meg. Egy sorban balra a kép (96 px), jobbra a cikkszám, név, leírás, ár, akció és a Rendelés gomb, köztük 1 px-es elválasztó vonal.
- A fix magasságok mobilon megszűnnek (a tégláknak nem kell egy vonalban lenniük), így a 2 soros nevek sem vágódnak le.
- Csak CSS és egy új osztály (`t-card`) a terméktégla belső táblázatán. Asztali, tablet (481 px felett) és Outlook nézet változatlan; média-lekérdezés nélküli kliensben továbbra is 3 oszlop.
- Érintett: minden sablon a `sablonok/` és `elonezet/` mappában.

## 1.2.1 · 2026-10-02 · v2 Waterside: téglaigazítás

**Mezők és hivatkozások: nincs változás** (ellenőrizve: a v2 kulcskészlete = a v5 kulcskészlete a `poll.*` és `cover.meta.poll` nélkül).
- `sablonok/v2-waterside.html`: az ár cellája 44 helyett 54 px magas, mint a v5-ben, így üres akciósornál is egy vonalban maradnak a téglák.
- Külön csomag: `export-v2` (csak a v2 sablon, a szabványos mappaszerkezetben).

## 1.2 · 2026-09-30 · Új sablon: v5 Ősz

**Mezők és hivatkozások: nincs változás.** A v5 pontosan a v4 kulcskészletét használja (a generáláskor ellenőrizve). A generátorban csak a sablonlistába kell felvenni a `v5-osz`-t. A v1, v2 és v4 sablon nem változott.

Új fájlok:
- `sablonok/v5-osz.html`, `elonezet/v5-osz.html`;
- `tartalom-v5-osz.json` (őszi mintatartalom, a `tartalom-minta.json` szerkezetével);
- `assets/leaf-edge-light.png`, `assets/leaf-edge-dark.png`, `assets/leaf-marker.png`;
- `assets/leaf-fall-a.png`, `assets/leaf-fall-b.png`, `assets/leaf-fall-c.png`, `assets/cover-osz-helyorzo.jpg`.

A v5 a v4 elrendezésére épül, a modulok és a sorrendjük ugyanaz. Ami más:
- **Színek:**
  - a lap háttere a rendszer Homok tónusa (#F2E8D8);
  - a kérdés-sáv Hajnal tónusú (#FCE6C8);
  - az elválasztó vonal meleg tónusú (#E6D3B5).
- **Levél-elemek (3 helyen):**
  - a borítósáv és a levél határán két juharlevél, a mélységtérkép-sáv helyén (egy kép, a sötét és a világos sávot is tartalmazza);
  - egy kis levél az ajánlat címkéje előtt;
  - az opcionális lehulló levelek a borítón.
- **Mozgás:** (b) CSS-kiegészítés, csak Apple Mail és iOS Mail, egyszer fut le, csökkentett mozgásnál nincs.
- **Sötét mód:** `prefers-color-scheme` és Outlook.com szabályok. Ez új a v4-hez képest, de mezőt nem érint.
- **Reszponzivitás:** ugyanazok a töréspontok és osztályok, mint az 1.1-ben.
- A `reszponziv-teszt.html` most mind a négy sablont mutatja.
- **Javítás a v5-ben:**
  - az ár cellája asztali nézetben 54 px magas (eddig 44 px), így az ár és az akciósor elfér, és üres akciósornál is egy vonalban maradnak a téglák;
  - a mintatartalom leírásai rövidebbek lettek;
  - a `desc` mező ajánlott hosszkorlátja 24 karakter lett (eddig 30).

## 1.1 · 2026-09-30 · Reszponzivitás (mobil + tablet)

**Mezők és hivatkozások: nincs változás.** Minden `{{kulcs}}` helyőrző ugyanaz, ugyanabban a sorrendben (a generálás közben ellenőrizve), a `tartalom-minta.json` és a `MEZOK-ES-GENERATOR.md` változatlanul érvényes. A generátoron nem kell módosítani.

Érintett fájlok: `sablonok/v1-sotet-lemez.html`, `sablonok/v2-waterside.html`, `sablonok/v4-partnerjelentes.html` (és az `elonezet/` változatuk).

### Töréspontok
| Szélesség | Viselkedés |
|---|---|
| 1025 px felett (asztali) | Változatlan: 600 px-es levél, 3 oszlopos termékrács |
| 641-1024 px (tablet) | 600 px középen, nagyobb külső margó (32/24 px) |
| 481-640 px (kis tablet, nagy telefon) | Folyékony szélesség, 3 oszlopos rács, kisebb oldalmargók (28 px) |
| 361-480 px (mobil) | 2 oszlopos rács (3 sor), egymás alá rendezett sorok, teljes szélességű képviselő-gombok |
| 360 px és alatta (kis mobil) | Mint a mobil, még kisebb oldalmargók (16 px) és főcím |

### Változások
- **Folyékony alap:** a 600 px-es fő táblázat `width:100%; max-width:600px`, így média-lekérdezés nélküli kliensekben is (pl. Gmail app nem Google-fiókkal) a képernyőhöz igazodik. Az Outlook asztali továbbra is a rejtett 600 px-es MSO-táblázatot használja.
- **Termékrács:** alapból 3 oszlop (33,333%), mobilon 2 oszlop (50%). Média-lekérdezés nélkül is legalább 3 oszlopban, szétcsúszás nélkül jelenik meg.
- **Terméktéglák mobilon:**
  - kisebb képdoboz (124 px, a kép legfeljebb 108 px);
  - a név 3 sornyi, a leírás 3 sornyi, az ár és az akció 64 px magas fix helyet kap, így a téglák keskenyebb oszlopban is egy vonalban maradnak;
  - a betűméret nem csökken, a szöveg mindenhol legalább 16 px.
- **Utility-sáv:** mobilon a szám felirata és a „Megnyitás böngészőben” link egymás alá kerül.
- **Ajánlat fejléce:** mobilon a címke és a jelvény („6 tétel · okt. 31-ig”) egymás alá kerül.
- **Képviselő:** mobilon a telefon- és e-mail-gomb egymás alatt, teljes szélességben jelenik meg, a hosszú e-mail-cím tördelhető.
- **Főcím mérete:**
  - v1 és v2: 28 px mobilon, 26 px kis mobilon;
  - v4: 40 px tableten, 34 px mobilon, 30 px kis mobilon.
- **Borító meta-sor** (szám · olvasási idő · termékek · kérdés): mobilon a tételek között tördelhető, így nem tolja szélesebbre a levelet a képernyőnél.
- **v4 lábléc:** mobilon a közösségi ikonok a logó alá kerülnek, balra igazítva.
- **Új tesztfájl:** az `elonezet/reszponziv-teszt.html` mindhárom sablont 360, 414 és 768 px szélességben mutatja egymás mellett.

### Nem változott
- A tervek, a színek, a betűtípusok és a tartalom.
- Az Outlook asztali megjelenés.
- Az `assets/` mappa.

---

## 1.0 · 2026-09-30 · Első kiadás
- Három táblázatos e-mail sablon (v1 Sötét lemez, v2 Waterside, v4 Partnerjelentés), közös `{{kulcs}}` mezőkkel.
- `tartalom-minta.json`, `MEZOK-ES-GENERATOR.md`, `assets/`.
