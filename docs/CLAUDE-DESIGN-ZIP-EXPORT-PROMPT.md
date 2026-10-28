# Feladat: ZIP-exportálási szabályok az Energofish hírlevélsablonokhoz

Szia! A sablonokat ZIP-csomagban töltjük be a generátoralkalmazásba (Sablon › „Sablon hozzáadása”). Az alkalmazás a csomagból automatikusan kiolvassa a sablonokat és a képeiket. Ahhoz, hogy ez minden alkalommal hibátlanul működjön, a ZIP-nek az alábbi szabályokat kell követnie. **Ez projektszabály, minden mostani és jövőbeli exportra érvényes.** Kérlek, írd be a saját dokumentációdba is (lásd a 7. pontot).

---

## 1. Csomagonként egy sablon, állandó szerkezettel

Minden sablon külön ZIP-be kerül, és a ZIP-ben **egyetlen gyökérmappa** van:

```
Energofish_Partner_Newsletter_Design_<azonosító>.zip
└── export-<azonosító>/
    ├── sablonok/
    │   └── <azonosító>.html        ← a sablon ({{kulcs}} helyőrzőkkel), PONTOSAN EGY fájl
    ├── elonezet/
    │   └── <azonosító>.html        ← a mintatartalommal kitöltött előnézet
    ├── assets/                     ← a sablon ÖSSZES képe, almappa nélkül
    │   ├── cover-….jpg
    │   └── ….png
    ├── tartalom-<azonosító>.json   ← mintatartalom
    ├── CHANGELOG.md
    ├── MEZOK-ES-GENERATOR.md
    ├── RESZPONZIV-SZABALYOK.md
    ├── EXPORT-SZABALYOK.md
    └── reszponziv-teszt.html
```

A mostani négy csomag azonosítója: `v1-sotet-lemez`, `v2-waterside`, `v6-tel`, `v9-kiarusitas`. A gyökérmappák neve: `export-v1`, `export-v2`, `export-v6-tel`, `export-v9-kiarusitas`.

## 2. A sablonfájl neve = az azonosító (ezt az alkalmazás azonosítónak használja)

- A fájlnév formája `v<szám>-<név>.html`: csak kisbetű (a–z), számjegy és kötőjel. **Nincs** ékezet, szóköz, nagybetű, aláhúzás vagy dátum. Például `v9-kiarusitas.html`, nem `V9 Kiárusítás.html`.
- **A meglévő sablon frissítésekor a fájlnév nem változhat.** Az alkalmazás a fájlnév alapján ismeri fel, hogy egy meglévő sablon új változatáról van szó, és felülírja azt. Más név esetén új, külön sablon jön létre.
- Új sablon új, még nem használt azonosítót kap. Ha a „v” után álló sorszám már foglalt, a név többi része tegye egyedivé (pl. `v6-tel` és `v6-osz-avar`).
- A sablonfájl a `sablonok/` mappában van, és ott **csak ez az egy** HTML lehet.

## 3. `{{` csak a sablonfájlban szerepelhet

Az alkalmazás a csomag minden olyan `.html` fájlját sablonnak tekinti, amelyben `{{` karaktersor van. Ezért:
- az `elonezet/` fájlja teljesen kitöltött, **egyetlen `{{` sincs benne**;
- a `reszponziv-teszt.html`-ben és minden más HTML-ben sincs `{{`, a magyarázó szövegekben és megjegyzésekben sem (`&#123;&#123;` vagy a kulcs kapcsos zárójel nélkül írható);
- a `.md` és `.json` fájlokban szerepelhet `{{` (ezeket az alkalmazás nem olvassa).

## 4. Képek

- A sablon minden képe így hivatkozik: `{{assets.base}}/<fájlnév>`. A fájlnév közvetlenül a perjel után áll, **almappa nélkül**, és csak `a–z`, `0–9`, `-`, `_`, `.` karakterekből állhat (ékezet, szóköz és nagybetű nélkül). Például `{{assets.base}}/tel-snow-band.png`.
- Minden hivatkozott kép **ott van** az `assets/` mappában, pontosan ugyanazzal a névvel (kis- és nagybetűre is egyezően). Ami nincs hivatkozva, az ne kerüljön a csomagba, kivéve a mintatartalom képeit (pl. a borító mintaképe).
- Formátum: **PNG vagy JPG**. WEBP, SVG és GIF nem lehet (az Outlook nem jeleníti meg őket).
- **Egy fájlnév mindig ugyanazt a képet jelenti**, minden csomagban. Ha egy kép tartalma megváltozik, új nevet kap (pl. `tel-edge-light-2.png`). A közös képek (`energofish-mark.png`, `energofish-mark-light.png`, `portre-helyettesito.png`, `social-*.png`) tartalma nem változik.
- Külső (http) képhivatkozás a sablonban nem lehet, minden kép az `assets/`-ből jön.

## 5. Fájlok és kódolás

- A HTML, a JSON és az MD fájlok **UTF-8** kódolásúak.
- Méretkorlátok:
  - a sablon HTML legfeljebb **90 KB** (a Gmail 102 KB felett levágja a kitöltött levelet);
  - a `<style>` blokk legfeljebb **16 KB** (a Gmail efölött a teljes stílust eldobja);
  - egy kép legfeljebb 1 MB.
- A ZIP-ben ne legyenek rejtett vagy rendszerfájlok: `.DS_Store`, `__MACOSX/`, `Thumbs.db`, `.git`. Ne legyen beágyazott ZIP, jelszóvédelem vagy szimbolikus link.
- Normál ZIP (Deflate), UTF-8 fájlnevekkel.

## 6. Ellenőrzőlista minden export előtt (írd a CHANGELOG-bejegyzés végére, kipipálva)

- [ ] A ZIP-ben egy gyökérmappa van: `export-<azonosító>/`.
- [ ] A `sablonok/` mappában pontosan egy HTML van, a neve `<azonosító>.html`, és megegyezik az előző változat nevével.
- [ ] Csak a sablonfájlban van `{{`, az `elonezet/` és a `reszponziv-teszt.html` fájlban nincs.
- [ ] Minden `{{assets.base}}/…` hivatkozott kép megvan az `assets/`-ben, egyező névvel; PNG vagy JPG.
- [ ] Nincs megváltozott tartalmú kép régi néven.
- [ ] A sablon ≤ 90 KB, a `<style>` ≤ 16 KB.
- [ ] A kulcskészlet változatlan (gépileg ellenőrizve).
- [ ] A `reszponziv-teszt.html` A–E próbái mind sikeresek (lásd `RESZPONZIV-SZABALYOK.md`).
- [ ] Nincs rejtett vagy rendszerfájl a ZIP-ben.

## 7. Dokumentáció

1. Új fájl a csomagban: **`EXPORT-SZABALYOK.md`**: a fenti 1–6. pont a saját szavaiddal. Kerüljön minden csomagba.
2. A `MEZOK-ES-GENERATOR.md`-ben a csomagszerkezetet bemutató rész legyen összhangban az 1. ponttal, és hivatkozzon az `EXPORT-SZABALYOK.md`-re.
3. `CHANGELOG.md`: rövid bejegyzés arról, hogy ettől a változattól az export ezeket a szabályokat követi.
4. Ha van projektszintű utasításod vagy emlékeztetőd, vedd fel: „Minden export az EXPORT-SZABALYOK.md szerint készül, és a végén kipipált ellenőrzőlista áll.”
