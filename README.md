# Energofish Partnerhírlevél-generátor

Windowsos asztali program, amely egy Excel-partnerlistából **partnerenként legenerálja a hírlevelet** a tervezett Energofish-sablonok (v1 Sötét lemez, v2 Waterside, v4 Partnerjelentés) valamelyikével.

- Ami partnerenként más (e-mail, név, területi képviselő és a fotója, elérhetőségei), az az **Excelből** jön.
- A termékek az Excel **második munkalapjáról** jönnek (cikkszám, cikknév, kép link, gomb link…).
- Minden más, ami minden partnernél ugyanaz (tárgy, borító, levél, ajánlat szövegei, kérdés, lábléc…), a **programban** adható meg.
- Gépelés közben **élő előnézet** mutatja a levelet bármelyik partnerre, asztali és mobil nézetben.

![Lépések: Adatok → Tartalom → Termékek → Ellenőrzés → Generálás](docs/kepernyo-adatok.png)

---

## Letöltés és indítás

1. Töltsd le a [`dist/EnergofishHirlevel.exe`](dist/EnergofishHirlevel.exe) fájlt (és ha kell, a [`dist/Energofish_partner_hirlevel_minta.xlsx`](dist/Energofish_partner_hirlevel_minta.xlsx) mintát).
2. Indítsd el dupla kattintással. Telepíteni nem kell, egyetlen fájl.

**Rendszerigény:** Windows 10 vagy 11. A program a Windowsba épített *Microsoft Edge WebView2* komponenssel jeleníti meg a felületét (Windows 11-en és frissített Windows 10-en megvan). Ha hiányzik, a program automatikusan az alapértelmezett böngészőben nyílik meg, és úgy is teljes értékűen működik.

> **„A Windows megvédte a számítógépet” üzenet:** a program nincs digitálisan aláírva, ezért az első indításnál a SmartScreen figyelmeztethet. Kattints a **További információ → Futtatás mindenképp** gombra.

---

## Használat 5 lépésben

| Lépés | Mit csinálsz |
|---|---|
| **1. Adatok** | Húzd be az Excelt az ablakba, vagy *Excel kiválasztása…*. Nincs még Excel? *Minta Excel mentése* – kész, kitöltött példa. A program kiírja, melyik oszlopot minek ismerte fel, és jelzi a hibás sorokat. A partnerek egyenként ki-be kapcsolhatók; egy sorra kattintva az előnézet arra a partnerre vált. |
| **2. Tartalom** | A közös szövegek, blokkonként (Alapadatok, Borító, Személyes levél, Ajánlat, Egy kérdés, Képviselő-blokk, Lábléc, Termék-linkek). A karakterszámláló a tervezői hosszkorlátokat figyeli. |
| **3. Termékek** | Az Excelből betöltött termékek: ki-be kapcsolás, sorrend, szövegjavítás, új termék. (Az Excel-fájlt nem módosítja.) |
| **4. Ellenőrzés** | Hibák és figyelmeztetések „Ugrás” gombbal a hibás mezőhöz; *Képek ellenőrzése* – letölti és méri a képeket. |
| **5. Generálás** | Sablon, kimeneti mappa, fájlnév-minta, EML-piszkozatok. Egy gomb, és kész. |

Minden beállítás automatikusan mentődik; a program a következő indításkor ugyanonnan folytatja (az utoljára használt Excelt is újraolvassa).

---

## Az Excel felépítése

Tetszőleges oszlopsorrend, a program a **fejléc szövegéből** ismeri fel az oszlopokat (kis-/nagybetű és ékezet nem számít, pl. „E-mail cím”, „Partner e-mail” egyaránt jó).

### 1. munkalap – Partnerek (soronként egy címzett)

| Oszlop | Kötelező | Mire kell |
|---|:-:|---|
| Partner e-mail | ✔ | A címzett. Hibás/hiányzó e-mail cím esetén a sor kimarad. |
| Partner neve | ✔ | Megszólítás: „Kedves {nev}!”. Üresen „Kedves Partnerünk!” lesz. |
| Területi képviselő | ✔ | A képviselő neve; ebből készül a monogram, ha nincs fotó. |
| Képviselő kép link | | https:// kép, négyzetes (min. 128×128). Üresen monogram jelenik meg. |
| Képviselő telefon | | Pl. `+36 30 123 4567` vagy `06301234567`. Üresen a telefongomb elmarad. |
| Képviselő e-mail | | A képviselő e-mail gombja. Üresen a gomb elmarad. |
| Képviselő területe | | A név alatt, pl. „Pest és Nógrád megye”. |
| Cégnév | | Fájlnévben, küldési listában, `{ceg}` változóként. |

**Extra lehetőségek**

- **Bármilyen további oszlop** változóként használható a szövegekben: a „Partnerkód” oszlop értéke a `{partnerkod}` helyére kerül.
- **Partnerenkénti felülírás:** ha egy oszlop fejléce sablonkulcs (pl. `note.body`, `cover.headline`, `offer.more.url`), az adott partnernél az ott megadott érték lép a közös szöveg helyére (üres cella = közös szöveg).

### 2. munkalap – Termékek (felülről lefelé ebben a sorrendben kerülnek a levélbe)

| Oszlop | Kötelező | Megjegyzés |
|---|:-:|---|
| Cikkszám | ✔ | |
| Cikknév | ✔ | max. kb. 26 karakter (2 sor) |
| Cikk kép link | | fehér hátterű, négyzetes, min. 260×260. Üresen: `https://images.energofish.hu/thumbimage/<cikkszám>.JPG` |
| Gomb link | | hová mutasson a gomb és a kép. Üresen: `https://b2b.energofish.hu/termek/<cikkszám>` |
| Rövid leírás | | max. kb. 30 karakter |
| Ár | | számként megadva „3 090 Ft” alakra formázódik |
| Akció | | max. kb. 18 karakter, üres is lehet |
| Aktív | | opcionális: „nem” értéknél a termék nem kerül a levélbe |

A rács asztali nézetben 3 oszlopos (mobilon 2), ezért **3, 6 vagy 9 termék** mutat a legjobban; 1–12 termék bármikor működik.
Linkeknél a cellához rendelt hivatkozás (Ctrl+K) és a `HYPERLINK()` képlet is működik.

---

## Változók

A közös szövegekben (tárgy, megszólítás, levél, linkek…) használhatók, partnerenként cserélődnek:

| Változó | Érték |
|---|---|
| `{nev}` | partner neve |
| `{email}` | partner e-mail címe |
| `{ceg}` | cégnév |
| `{kepviselo}` | területi képviselő neve |
| `{kepviselo_email}`, `{kepviselo_telefon}`, `{terulet}` | a képviselő elérhetőségei |
| `{termekszam}` | a levélben szereplő termékek száma (pl. „{termekszam} termék”) |
| `{assets}` | a képtár (assets) webcíme |
| `{bármely-oszlop}` | az Excel bármely további oszlopa (fejléc ékezet és szóköz nélkül) |

Linkekben az értékek automatikusan URL-kódolódnak, pl. a kérdés válaszlinkje: `https://energofish.hu/kerdes/2026-10?valasz=feeder&partner={email}`.

---

## Kimenet

Minden generálás egy új, dátummal jelölt mappába kerül a kimeneti mappán belül:

```
2026-10-01_09-30_v4-partnerjelentes\
  attekinto.html          ← áttekintő: minden partner, egy kattintással megnyitható levelek
  kuldesi-lista.csv       ← e-mail, név, cég, képviselő, tárgy, fájl (Excelben megnyitható)
  tartalom.json           ← a felhasznált közös tartalom (archívum, újra betölthető)
  html\001_partner@cim.hu.html   ← a kész levelek, partnerenként
  eml\001_partner@cim.hu.eml     ← (ha kéred) Outlookban megnyitható, elküldhető piszkozat
  assets\                 ← (ha nincs megadva képtár-cím) a sablon képei
```

- A **HTML** fájlok a küldőrendszerbe (Mailchimp, SendGrid, saját rendszer…) tölthetők be.
- Az **EML** fájl dupla kattintással Outlookban nyílik meg, címzettel, tárggyal kitöltve, azonnal elküldhető (HTML + egyszerű szöveges változattal).
- A fájlnév mintája állítható (`{sorszam}`, `{email}`, `{nev}`, `{ceg}`, `{kepviselo}`).

### Kiküldés előtt: a képtár

A sablonok saját képei (logó, hullámok, mélységtérkép, alapértelmezett borító és portré) az [`assets/`](assets) mappában vannak. **Kiküldés előtt** ezt a mappát fel kell tölteni egy nyilvános https tárhelyre, és a címét megadni a *Tartalom → Alapadatok → Képtár (assets mappa) webcíme* mezőben. Amíg ez üres, a program a kimenet mellé másolja a képeket: helyben minden jól látszik, de a címzettnél nem.

---

## Sablonok

A tervezői sablonok (`docs/MEZOK-ES-GENERATOR.md`, `{{kulcs}}` helyőrzők) a [`sablonok/`](sablonok) mappában vannak, a következő kiegészítésekkel – alapesetben a kimenet **bájtra azonos** a tervezői előnézettel (ezt teszt ellenőrzi):

- a 6 fix terméktégla helyett 1–12 termék (Outlookban is 3 oszlopos rács);
- **képviselő fotója** kör alakban, narancs kerettel; ha nincs fotó, a monogram;
- a telefon- és e-mail-gomb, a „teljes ajánlat” link, a webes verzió linkje és a közösségi ikonok üres érték esetén elmaradnak;
- v4: az „Egy kérdés” blokk üres kérdésnél elmarad, a preheaderhez automatikusan hozzáfűzi a „Plusz egy kérdés.” szöveget.

Új sablonverzió esetén: a tervezői fájlokat az `internal/hirlevel/testdata/eredeti/` mappába kell tenni, majd `python3 tools/sablon_atalakitas.py`.

---

## Hibaelhárítás

| Jelenség | Megoldás |
|---|---|
| SmartScreen figyelmeztetés | *További információ → Futtatás mindenképp.* |
| Böngészőben nyílik meg ablak helyett | Hiányzik a WebView2: [telepíthető a Microsofttól](https://developer.microsoft.com/microsoft-edge/webview2/), de böngészőben is minden működik. Kilépés: jobb felső menü → *Kilépés a programból*. |
| Az Excel módosítása nem látszik | Mentsd a fájlt Excelben, majd *Újratöltés*. |
| „A régi .xls formátum nem támogatott” | Excelben *Mentés másként → Excel-munkafüzet (.xlsx)*. |
| A termékképek nem látszanak | Az *Ellenőrzés → Képek ellenőrzése* megmutatja, melyik link rossz. |

Beállítások és napló: `%APPDATA%\EnergofishHirlevel\` (`beallitasok.json`, `naplo.txt`) – a program menüjéből is megnyitható.

---

## Parancssor (haladóknak)

```
EnergofishHirlevel.exe -batch -excel partnerek.xlsx -sablon v4 -kimenet D:\Hirlevel [-tartalom tartalom.json] [-eml] [-felado "Energofish <hirlevel@energofish.hu>"]
```

Felület nélkül generál (pl. ütemezett feladatból); a közös tartalom a mentett beállításokból vagy a `-tartalom` JSON-ból jön, az eredmény a naplóba kerül. `-bongeszo`: saját ablak helyett böngészőben nyílik meg.

---

## Fejlesztőknek

Go 1.25+, külső futtatókörnyezet nélkül; a felület beágyazott HTML/CSS/JS (Open Sans betűkészlettel), Windowson WebView2 ablakban.

```
./build.sh 1.0.0        # tesztek + dist/EnergofishHirlevel.exe (Linuxról is fordítható)
go test ./...           # egység- és API-tesztek
go run . -bongeszo      # fejlesztői futtatás böngészőben
```

| Hely | Tartalom |
|---|---|
| `main.go`, `app.go` | indítás, helyi HTTP API (csak 127.0.0.1, tokennel védve) |
| `platform_windows.go` | WebView2 ablak, natív fájlválasztók, DPI-kezelés |
| `internal/hirlevel/` | sablonmotor, Excel-olvasó, validálás, generálás (HTML, EML, CSV) |
| `web/` | a felület |
| `sablonok/`, `assets/` | hírlevélsablonok és képeik |
| `demo/`, `tools/minta_excel.py` | minta Excel és előállító szkriptje |
| `winres/` | ikon, manifest, verzióinfó (`go-winres make --in winres/winres.json --out rsrc`) |

A `.github/workflows/windows-build.yml` minden pushnál Windows gépen fordít, lefuttatja a teszteket, az öntesztet és egy valódi WebView2-ablakos tesztet, az exe-t pedig letölthető artefaktként csatolja.
