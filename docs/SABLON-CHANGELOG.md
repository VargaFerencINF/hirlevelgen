# Changelog: Energofish hírlevél-sablonok

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
