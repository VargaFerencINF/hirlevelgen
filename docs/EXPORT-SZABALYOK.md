# Export: kötelező ZIP-szabályok (projektszabály)

A sablonokat ZIP-csomagban töltjük be a generátoralkalmazásba (Sablon › „Sablon hozzáadása”). Az alkalmazás a csomagból magától olvassa ki a sablont és a képeit. **Minden mostani és jövőbeli export e szerint készül, és a CHANGELOG-bejegyzés végén kipipált ellenőrzőlista áll** (6. pont).

---

## 1. Egy ZIP = egy sablon, mindig ugyanazzal a szerkezettel

Minden sablon külön ZIP-be kerül. A ZIP-ben **pontosan egy gyökérmappa** van, más semmi:

```
Energofish_Partner_Newsletter_Design_<azonosító>.zip
└── export-<…>/
    ├── sablonok/
    │   └── <azonosító>.html        ← a sablon ({{kulcs}} helyőrzőkkel), pontosan egy fájl
    ├── elonezet/
    │   └── <azonosító>.html        ← a mintatartalommal teljesen kitöltött előnézet
    ├── assets/                     ← a sablon összes képe, almappa nélkül
    │   ├── cover-….jpg
    │   └── ….png
    ├── tartalom-<azonosító>.json   ← mintatartalom
    ├── CHANGELOG.md
    ├── MEZOK-ES-GENERATOR.md
    ├── RESZPONZIV-SZABALYOK.md
    ├── EXPORT-SZABALYOK.md
    └── reszponziv-teszt.html
```

A jelenlegi csomagok (a gyökérmappa neve állandó, nem változik):

| Sablon | Azonosító (= sablonfájl neve) | Gyökérmappa | ZIP |
|---|---|---|---|
| v1 Sötét lemez | `v1-sotet-lemez` | `export-v1/` | `Energofish_Partner_Newsletter_Design_v1-sotet-lemez.zip` |
| v2 Waterside | `v2-waterside` | `export-v2/` | `Energofish_Partner_Newsletter_Design_v2-waterside.zip` |
| v6 Ősz (avar) | `v6-osz-avar` | `export-v6/` | `Energofish_Partner_Newsletter_Design_v6-osz-avar.zip` |
| v6 Tél | `v6-tel` | `export-v6-tel/` | `Energofish_Partner_Newsletter_Design_v6-tel.zip` |
| v9 Kiárusítás | `v9-kiarusitas` | `export-v9-kiarusitas/` | `Energofish_Partner_Newsletter_Design_v9-kiarusitas.zip` |

## 2. A sablonfájl neve az azonosító

- Forma: `v<szám>-<név>.html`. Csak kisbetű (a–z), számjegy és kötőjel; **nincs** ékezet, szóköz, nagybetű, aláhúzás vagy dátum. Helyes: `v9-kiarusitas.html`; hibás: `V9 Kiárusítás.html`, `v9_kiarusitas_2026-10.html`.
- **Frissítéskor a fájlnév nem változhat.** Az alkalmazás a fájlnévből ismeri fel, hogy egy meglévő sablon új változata érkezett, és felülírja. Más névvel új, külön sablon jönne létre.
- Új sablon új, még nem használt azonosítót kap. Ha a „v” utáni sorszám foglalt, a név többi része teszi egyedivé (pl. `v6-tel` és `v6-osz-avar`, `v9-black-friday-grafit` és `v9-kiarusitas`).
- A sablon a `sablonok/` mappában van, és ott **csak ez az egy** HTML lehet.
- Az előnézet (`elonezet/<azonosító>.html`) és a mintatartalom (`tartalom-<azonosító>.json`) ugyanazt az azonosítót viseli.

## 3. Kapcsos zárójelpár csak a sablonfájlban

Az alkalmazás minden olyan `.html` fájlt sablonnak tekint, amelyben két nyitó kapcsos zárójel áll egymás után. Ezért:
- az `elonezet/` fájlja teljesen ki van töltve, egyetlen helyőrző sem marad benne;
- a `reszponziv-teszt.html`-ben és minden más HTML-ben sincs ilyen karaktersor, a magyarázó szövegben, a megjegyzésben és a szkriptben sem (helyette `&#123;&#123;`, vagy a kulcs kapcsos zárójel nélkül);
- a `.md` és a `.json` fájlokban szerepelhet (ezeket az alkalmazás nem olvassa).

## 4. Képek

- A sablon minden képe így hivatkozik: `{{assets.base}}/<fájlnév>`. A fájlnév közvetlenül a perjel után áll (**nincs almappa**), és csak `a–z`, `0–9`, `-`, `_`, `.` karaktert tartalmaz. Példa: `{{assets.base}}/tel-snow-band.png`.
- **Külső (http/https) képhivatkozás a sablonban nincs**, minden sablonkép az `assets/` mappából jön. (A termékképek mezőtartalmak: `{{offer.items.N.image}}`, ezek URL-jét a tartalom adja.)
- Minden hivatkozott kép megvan az `assets/` mappában, pontosan ugyanazzal a névvel (kis- és nagybetűre is). Ami nincs hivatkozva, nem kerül a csomagba, kivéve a mintatartalom képeit (borító-mintakép, helyettesítő portré).
- Formátum: **PNG vagy JPG**. WEBP, SVG és GIF nem lehet (az Outlook nem jeleníti meg).
- **Egy fájlnév mindig ugyanazt a képet jelenti, minden csomagban, bájtra azonosan.** Ha egy kép tartalma megváltozik, új nevet kap (pl. `tel-edge-light-2.png`); a régi néven nem cserélhető le.
- A közös képek tartalma soha nem változik, és minden csomagban ugyanaz a fájl:
  - `energofish-mark.png`, `energofish-mark-light.png` (logó, sötét és világos változat);
  - `portre-helyettesito.png`;
  - `social-facebook.png`, `social-youtube.png`, `social-instagram.png`, `social-tiktok.png`;
  - `cover-balaton-1200x660.jpg` (minta-borítókép).

## 5. Fájlok, kódolás, méret

- HTML, JSON és MD: **UTF-8**.
- Méretkorlátok:
  - sablon HTML: legfeljebb **90 KB** (a Gmail 102 KB felett levágja a kitöltött levelet);
  - `<style>` blokk: legfeljebb **16 KB** (efölött a Gmail a teljes stílust eldobja);
  - egy kép: legfeljebb **1 MB**.
- Nincs rejtett vagy rendszerfájl: `.DS_Store`, `__MACOSX/`, `Thumbs.db`, `.git`. Nincs beágyazott ZIP, jelszó és szimbolikus link.
- Normál ZIP (Deflate), UTF-8 fájlnevekkel (a ZIP „UTF-8 név” jelzőjével).

## 6. Ellenőrzőlista minden export előtt

A CHANGELOG-bejegyzés végére, kipipálva:

- [ ] A ZIP-ben egy gyökérmappa van: `export-<…>/` (a fenti táblázat szerinti név).
- [ ] A `sablonok/` mappában pontosan egy HTML van, a neve `<azonosító>.html`, és megegyezik az előző változat nevével.
- [ ] Csak a sablonfájlban van kapcsos zárójelpár, az `elonezet/` és a `reszponziv-teszt.html` fájlban nincs.
- [ ] Minden `{{assets.base}}/…` hivatkozott kép megvan az `assets/`-ben, egyező névvel; PNG vagy JPG; nincs külső képhivatkozás.
- [ ] Nincs megváltozott tartalmú kép régi néven.
- [ ] A sablon ≤ 90 KB, a `<style>` ≤ 16 KB.
- [ ] A kulcskészlet változatlan (gépileg ellenőrizve).
- [ ] A `reszponziv-teszt.html` A–E próbái mind sikeresek (lásd `RESZPONZIV-SZABALYOK.md`).
- [ ] Nincs rejtett vagy rendszerfájl a ZIP-ben.

A pontokat gépileg ellenőrizzük: a kész ZIP-et visszaolvassuk (bejegyzések, gyökérmappa, CRC), és a fenti feltételeket a ZIP tartalmán futtatjuk le, nem a munkamappán.
