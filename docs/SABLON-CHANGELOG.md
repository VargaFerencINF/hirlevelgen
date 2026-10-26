# Changelog: Energofish hírlevél-sablonok

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
