# Energofish Partnerhírlevél-generátor

Windowsos asztali program, amely egy partnerlistából **partnerenként legenerálja a hírlevelet** a tervezett Energofish-sablonok (v1 Sötét lemez, v2 Waterside, v4 Partnerjelentés) valamelyikével.

- Ami partnerenként más (e-mail, név, területi képviselő és a fotója, elérhetőségei), az a **B2B partnertörzsből** (célcsoport, képviselő, megye, besorolás… szerint összeállított halmaz, 1.3) vagy **Excelből** jön.
- A termékek az Excel **második munkalapjáról** jönnek (cikkszám, cikknév, kép link, gomb link…).
- Minden más, ami minden partnernél ugyanaz (tárgy, borító, levél, ajánlat szövegei, kérdés, lábléc…), a **programban** adható meg.
- Gépelés közben **élő előnézet** mutatja a levelet bármelyik partnerre, asztali és mobil nézetben.

![Lépések: Adatok → Tartalom → Termékek → Ellenőrzés → Generálás](docs/kepernyo-adatok.png)

---

## Letöltés és indítás

1. Töltsd le a legfrissebb kiadást a [Releases](https://github.com/VargaFerencINF/hirlevelgen/releases/latest) oldalról (vagy a [`dist/EnergofishHirlevel.exe`](dist/EnergofishHirlevel.exe) fájlt) (és ha kell, a [`dist/Energofish_partner_hirlevel_minta.xlsx`](dist/Energofish_partner_hirlevel_minta.xlsx) mintát).
2. Indítsd el dupla kattintással. Telepíteni nem kell, egyetlen fájl.

![A program Windows 11-en, kis képernyőn (1024×768) teljes méretre nyitva](docs/kepernyo-windows.png)

**Rendszerigény:** Windows 10 vagy 11. A program a Windowsba épített *Microsoft Edge WebView2* komponenssel jeleníti meg a felületét (Windows 11-en és frissített Windows 10-en megvan). Ha hiányzik, a program automatikusan az alapértelmezett böngészőben nyílik meg, és úgy is teljes értékűen működik.

> **„A Windows megvédte a számítógépet” üzenet:** a program nincs digitálisan aláírva, ezért az első indításnál a SmartScreen figyelmeztethet. Kattints a **További információ → Futtatás mindenképp** gombra.

---

## Használat 6 lépésben

| Lépés | Mit csinálsz |
|---|---|
| **1. Adatok** | *Partnerhalmaz összeállítása…* a B2B partnertörzsből (lásd lent), vagy húzd be az Excelt az ablakba / *Excel kiválasztása…*. Nincs még Excel? *Minta Excel mentése* – kész, kitöltött példa. A program kiírja, melyik oszlopot minek ismerte fel, és jelzi a hibás sorokat. A partnerek egyenként ki-be kapcsolhatók; egy sorra kattintva az előnézet arra a partnerre vált. |
| **2. Tartalom** | A közös szövegek, blokkonként (Alapadatok, Borító, Személyes levél, Ajánlat, Egy kérdés, Képviselő-blokk, Lábléc, Termék-linkek). A karakterszámláló a tervezői hosszkorlátokat figyeli. |
| **3. Termékek** | Az Excelből betöltött termékek: ki-be kapcsolás, sorrend, szövegjavítás. *Új termék:* keresés a friss cikktörzsben cikkszámra vagy névre, a mezők kitöltésével (lásd lent). Az Excel-fájlt nem módosítja. |
| **4. Ellenőrzés** | Hibák és figyelmeztetések „Ugrás” gombbal a hibás mezőhöz; *Képek ellenőrzése* – letölti és méri a képeket. |
| **5. Generálás** | Sablon, kimeneti mappa, fájlnév-minta, EML-piszkozatok. Egy gomb, és kész. |
| **6. Küldés** | Kiküldés Postmarkon: beállítások, kampánynév, mód (validálás → sandbox → belső teszt → éles), ellenőrzés, küldés, napló (lásd lent). |

Minden beállítás automatikusan mentődik; a program a következő indításkor ugyanonnan folytatja (az utoljára használt Excelt is újraolvassa).

---

## Partnerek a B2B partnertörzsből (1.3)

A webshop feliratkozói célcsoportonként (B2B HU, SK, CZ, COM, AT, DE, RO, ES, RS) külön JSON-exportból jönnek.
Az *Adatok* lépés **B2B partnertörzs** kártyáján:

1. **Források** – a linkeket egyszer kell megadni, utána a program titkosítva megjegyzi őket:
   - a legegyszerűbb: tedd a program mellé (vagy a beállítások mappájába) a `partnerforrasok.txt` fájlt, soronként `B2B HU: https://…&token=…` alakban. Indításkor a program beolvassa, titkosítva elmenti, és **a nyílt szöveges fájlt törli**; vagy húzd be a .txt-t az ablakba;
   - vagy a **Források…** gombnál illeszd be a célcsoportok tokenes linkjeit (pl. több sor `B2B HU: https://…&token=…` alakban; a program felismeri, melyik melyik), vagy célcsoportonként a tokent. A tokenek titkosan, a Windows-felhasználóhoz kötve (DPAPI) tárolódnak `%APPDATA%\EnergofishHirlevel\` alatt; a program sehol nem írja ki őket (a felületen is csak az első és utolsó 4 karakterük látszik), és a naplóba sem kerülnek. Környezeti változóból is megadhatók: `WEBGALAMB_TOKEN_B2B_HU`, `WEBGALAMB_TOKEN_B2B_SK`, …
2. **Partnerhalmaz összeállítása…** – célcsoport (ország) választás, **Frissítés most** (letöltés), majd a feltételek:
   - **területi képviselő**, **besorolás** (Basic … Top, Bizományos), **partnerbolt / horgászbolt**, **megye**, *Tulajdonság 6* – több érték is kijelölhető, és mindegyik megfordítható (*kivéve*);
   - **bizományosok** és **belső másolati címek** (Fix): mind / csak ők / nélkülük;
   - **feliratkozás dátuma** (-tól, -ig), **keresés** névre, e-mailre, Nazonra;
   - egyes partnerek **egyenként kizárhatók** a listából (és visszavehetők).

   Egy szemponton belül bármelyik, a szempontok között mindegyik feltételnek teljesülnie kell. Minden érték mellett látszik, hány partner felel meg rá a többi feltétellel együtt. A halmaz **elmenthető névvel**, és később egy kattintással visszatölthető.
   **Országtérkép:** a halmaz száma mellett a célcsoport országának kis térképe látszik, megyénként a partnerek számával (minél sötétebb, annál több partner). **Kattintással szűrhetsz** egy megyére (újabb kattintás: ki), a kijelölt megyék kiemelve, a kiszűrtek szürkén látszanak, alatta pedig a felirat mutatja, mire szűrtél („Szűrés: Pest, Nógrád”, „kivéve” esetén „Kivéve: …”). A térkép és a bal oldali Megye-lista ugyanazt a szűrőt állítja. A betöltött halmaz kártyáján (Adatok fül) is látszik a halmaz térképe.
   Minden célcsoportnak van térképe: Magyarország (19 megye + Budapest), Szlovákia, Csehország, Ausztria, Németország, Románia, Spanyolország (tartományok; a Kanári-szigetek keretben), Szerbia (körzetek). A partnertörzs Megye-értékeit a program név szerint párosítja a térkép régióival – helyi, magyar, angol és német névalakkal is („Bratislavský kraj”, „Pozsony”, „Bratislava”; „Hargita” / „Harghita”; „Bécs” / „Wien”), az összetett értékeknél a legpontosabb részre („ANDALUCÍA - ALMERÍA” → Almería; a javított, csak tartományt tartalmazó „Almería” is működik), és egy név több régiót is jelölhet („Vajdaság”, „Cataluña”). Ami nem párosítható (pl. „Külföld”), vagy aminél nincs megye megadva, a térkép alatt „Nincs a térképen” felsorolásban látszik, és a listában továbbra is választható.
   **Földgömb (B2B COM):** a nemzetközi célcsoportnál lassan forgó földgömb látszik a világ országaival. Egérrel forgatható, az országok fölött súgó mutatja a nevet és a partnerek számát. Ha kijelölsz egy országot (a földgömbön vagy a listában), a gömb odafordul, rázoomol, egy pillanatra megmutatja a nevével, majd visszaáll a teljes bolygóra. (Ha a Windowsban a mozgáscsökkentés be van kapcsolva, a gömb nem forog és nem animál.)
   A partnertörzs a böngészőben letöltött exportból (JSON) is betölthető: húzd be a fájlt az ablakba (a program megkérdezi, melyik célcsoporté), vagy a halmazválasztó **JSON-fájl…** gombja. Ugyanazok a szabályok érvényesek, mint a letöltésnél; régi fájlnál figyelmeztet (az azóta leiratkozottak újra aktívvá válnának).
3. **Betöltés a hírlevélhez** – a halmaz partnerlistaként töltődik be, a program többi része (tartalom, termékek, ellenőrzés, előnézet, generálás) ugyanúgy működik, mint Excellel.

**Mi kerül a levélbe:** ugyanazok a változók, mint az Excelből – a partnertörzs mezői így kerülnek a helyükre:

| Partnertörzs mező | Változó / levélrész |
|---|---|
| `Email_cim` | `{email}`, a címzett |
| `Nev` | `{nev}` **és** `{ceg}` (a csupa nagybetűs nevek olvasható írásmóddal: „JDB HUNGARY ZRT.” → „JDB Hungary Zrt.” – kikapcsolható); eredeti írásmóddal `{partnernev}` |
| `Teruleti_kepviselo_nev` / `_telefonszam` / `_email_cim` | `{kepviselo}` (a „ - Energofish Kft.” utótag nélkül – átállítható), `{kepviselo_telefon}`, `{kepviselo_email}`, a képviselő-blokk |
| `Megye` | `{terulet}` és a képviselő-blokk területe („Pest megye”, „Budapest”), valamint `{megye}` |
| `Teruleti_kepviselo_monogram` | a **képviselő fotója** (a partnertörzsben nincs: monogramonként a *Képviselő-fotók…* gombnál adható meg, nélküle monogram), `{tkmonogram}` |
| `Leiratkozas_link` | a lábléc leiratkozó linkje helyén **a partner saját leiratkozó linkje** |
| `Nazon`, `Besor`, `Partnerbolt_statusz`, `Telefonszam`, `Feliratkozas_datum` | `{nazon}`, `{besorolas}`, `{partnerbolt}`, `{telefon}`, `{feliratkozas}`; továbbá `{bizomanyos}`, `{celcsoport}` és az export esetleges új mezői |

A **megszólítás** a *Tartalom › Megszólítás* mező szerint („Kedves {nev}!”) a partner nevével készül; a halmaz-ablak *A levélbe kerülő adatok* részében átállítható, hogy a cégek (csupa nagybetű vagy Kft., Bt., Zrt. …) vagy mindenki a tartalék „Kedves Partnerünk!” megszólítást kapja – a többi változó ettől kitöltve marad. A betöltött lista saját változói a *Tartalom* fülön a változó-listában és a „Változó beszúrása” sávban is megjelennek.

**Szinkron (a partnertörzs saját, helyi adatbázisa):**

- az export lehet lista (`[{…},{…}]`) vagy számozott kulcsú objektum (`{"0":{…},"1":{…},"3":{…}}` – a Webgalamb így írja ki, ha egy rekordot kiszűr a lista közepéről); a program mindkettőt kezeli, a rekordokat a sorszámok szerint olvassa, a lyukakat átugorja. Csak az `Email_cim` kötelező: e-mail cím nélküli rekord figyelmeztetéssel kimarad, a hiányzó vagy üres többi mező nem hiba. Ugyanaz az e-mail cím kétszer: egy partner lesz belőle (egy levél), az Ellenőrzés lépésben látszik, melyik rekord maradt ki.
- új feliratkozó → bekerül; meglévő → frissül; aki újra feliratkozott → újra aktív; **aki kimaradt az exportból (leiratkozott vagy törölték) → inaktív**, és levelet nem kap. Partner soha nem törlődik, minden futás naplózódik (csak darabszámokkal).
- az exporton belül többször szereplő e-mail egy partnerré vonódik össze; az érvénytelen rekordok kimaradnak; a „Torolt” tokenű partnerek token nélkül szerepelnek.
- **biztonsági zár:** ha a letöltés hibás, üres, nem JSON, vagy az export a jelenlegi aktív partnerek 80%-ánál kevesebbet tartalmaz, **semmi nem változik** (a program figyelmeztet; tudatos döntéssel kényszeríthető).
- **generálás előtt a program mindig frissít:** a közben leiratkozottak kimaradnak, a feltételeknek megfelelő új feliratkozók bekerülnek. Ha a frissítés nem sikerül, a program megkérdezi, generáljon-e a legutóbb letöltött adatokkal.

**A leiratkozó linket a program soha nem nyitja meg** (egy megnyitás azonnal leiratkoztatná a partnert): az előnézetben helyettesítő link szerepel, a „Megnyitás” és a link-ellenőrzés tiltja, az áttekintő oldal figyelmeztet. A kész levelekben a valódi link van – ott se kattints rá.

A partner **tokenje** (automatikus bejelentkezés) titkosan tárolódik, de a linkekbe még **nem** kerül: a webes oldal elkészülte és a link-szabályok (paraméternév, csak energofish.hu linkek, UTM-ek) véglegesítése után kapcsolható be.

## Küldés Postmarkon (1.4)

A 6. lépés a kész, partnerenkénti leveleket a [Postmark](https://postmarkapp.com) REST API-ján küldi ki. A Postmark semmit nem renderel: minden partner pontosan azt a HTML-t kapja, amit a generálás is készít (ugyanaz a sablon és tartalom), csak a képek a **feltöltött képtárból** jönnek (a képtár webcíme kötelező, `cid:` vagy helyi kép nem lehet a levélben). A korábbi lépések és a Generálás nem változtak.

**Postmark beállítások** (*Küldés › Postmark beállítások…*):

| Beállítás | Mire való |
|---|---|
| Éles szerver API token | A Postmark szerver (pl. „My First Server”) *API Tokens* oldaláról. |
| Sandbox szerver API token | Opcionális: egy *Sandbox* típusú Postmark-szerver tokenje (nem kézbesít, a levelek a Postmark felületén látszanak). |
| Üzenetfolyam (stream) | A hírlevelek broadcast streamje, alapból `broadcast`. |
| Feladó | Alapból `Energofish Partner Brief <hirlevel@energofish.hu>` (a domain DKIM-mel és Return-Pathszal ellenőrzött). |
| Belső tesztcímek, tesztlevelek száma | A *Belső teszt* címzettjei; mindegyik ennyi különböző partner levelét kapja. |
| Link-kiegészítés (UTM) | Alapból `utm_source=partnerbrief&utm_medium=email&utm_campaign={kampany}`: minden kimenő http(s) linkhez hozzáfűzve – a leiratkozó link, a `mailto:`/`tel:` és a már `utm_source`-ot tartalmazó linkek kivételével. |
| Megnyitások mérése, linkkövetés | `TrackOpens` (alapból be), `TrackLinks` (alapból `None` – a Postmark linkkövetése csak a fiók jóváhagyása után működik). |
| Válaszcím: a területi képviselő | `ReplyTo` = a partner képviselőjének e-mail címe (képviselő nélkül a feladó). |
| Egykattintásos leiratkozás | `List-Unsubscribe-Post: List-Unsubscribe=One-Click` – csak akkor hagyd bekapcsolva, ha az energofish.hu leiratkozó oldala a POST kérést is kezeli (a Gmail „Leiratkozás” gombja így hívja meg). |

A tokenek a beállítások mappájában, **titkosítva** tárolódnak (`postmark-tokenek.dat`, Windowson a felhasználói fiókhoz kötve); a többi beállítás a `postmark.json`-ban. A token soha nem kerül a forráskódba, az exe-be, naplóba, CSV-be vagy hibaüzenetbe, a felület csak a maszkolt alakot mutatja.

**A négy mód:**

1. **Validálás** (alapértelmezett) – `POSTMARK_API_TEST` tokennel: a Postmark ellenőrzi a kéréseket, de semmit nem küld el.
2. **Sandbox** – a Sandbox szerverre; csak ha van Sandbox token.
3. **Belső teszt** – az éles szerverről, de csak a tesztcímekre, `[TESZT]` tárggyal.
4. **Éles** – a partnereknek. Csak „Custom” leiratkozás-kezelésű streamre küldhető, és a megerősítéshez be kell írni a címzettek számát.

Teszt módokban (1–3) a levelekben a partner **valódi leiratkozó linkje helyett egy ártalmatlan helyettesítő link** áll (a fejlécben is), így egy tesztelő kattintása nem iratkoztathat le senkit. A program a leiratkozó linkeket soha nem hívja meg.

**Ellenőrzés** (semmit nem küld): B2B halmaznál frissíti a partnertörzset; lekéri a stream beállítását (`UnsubscribeHandlingType`) és a letiltott címeket (`suppressions/dump`); összeállítja a leveleket, és összesíti a címzetteket, a kimaradókat (okkal) és a kötegek számát. Kimarad: a hibás vagy hiányzó cím, az ismétlődő cím, a Postmarkban letiltott cím (végleges visszapattanás, spamjelzés, Postmarkos leiratkozás), az Excel *Leiratkozott* oszlopában jelölt partner (igen / x / 1), az importált leiratkozott-listán szereplő cím, illetve ahol nincs érvényes leiratkozó link. Figyelmeztet a 100 KB-nál nagyobb levelekre (a Gmail levágja) és a képviselő nélküli partnerekre; letölti a képeket is (élesen az elérhetetlen kép megállítja a küldést).

- **„Custom” stream:** minden levél `List-Unsubscribe: <a partner saját leiratkozó linkje>` fejlécet kap (ugyanaz a link, mint a láthatóban) és – ha be van kapcsolva – `List-Unsubscribe-Post` fejlécet.
- **„Postmark” stream** (a fiók jóváhagyásáig): saját fejléc nincs (a Postmark a sajátját teszi be és egy saját leiratkozó linket is a levél aljára) – ez teszt módokban figyelmeztetés, élesen tiltott.

**Küldés:** `POST /email/batch`, legfeljebb 500 levél kötegenként, a kötegek egymás után (egyszerre egy kapcsolat). Levelenként: From, To, ReplyTo, Subject, HtmlBody, TextBody (a levél szöveges változata), MessageStream, Tag (= kampány), Metadata (`partner_id`, `campaign`, `nazon`), TrackOpens, TrackLinks. A HTTP 200 nem jelenti, hogy minden levél kiment: a program levelenként nézi az `ErrorCode`-ot, és magyarul írja ki a hibát (pl. *„a fiók még teszt módban van: csak @energofish.hu címekre lehet küldeni”*). 429 és 5xx válasznál növekvő várakozással újrapróbál; egy elküldött, de válasz nélkül maradt köteget soha nem küld újra vakon („bizonytalan” állapot – nézd meg a Postmark *Activity* oldalán). A *Megszakítás* a folyamatban lévő köteget még befejezi.

**Napló és folytatás:** kampányonként és módonként egy CSV a kimeneti mappában: `postmark-naplo\<kampány>_<mód>.csv` (UTF-8 BOM, Excelben megnyitható; oszlopok: `partner_id, email, rep_email, MessageID, ErrorCode, Message, mode, timestamp`). Ez az igazság forrása: ugyanazzal a kampánynévvel újraindítva a már sikeresen elküldött partnerek kimaradnak, így egy összeomlás, megszakítás vagy ismételt indítás sem küld duplán. Teszt módok naplója törölhető (*Teszt-napló törlése*), az élesé nem.

**Visszajelzések frissítése:** lekéri a Postmark letiltott címeit, és CSV-be menti (`postmark-visszajelzesek_<dátum>.csv` a kimeneti mappában) a betöltött partnerek adataival – az Excel / partnertörzs frissítéséhez. **Leiratkozottak importálása:** bármilyen CSV vagy szövegfájl, amiben e-mail címek vannak; ezek a címek ezután soha nem kapnak levelet.

**A Postmark fiók teszt módjában** (jóváhagyásig) összesen legfeljebb 100 levél küldhető, és csak `@energofish.hu` címekre – ilyenkor a *Belső teszt* használható, az éles küldés a Postmark jóváhagyása és a „Manage unsubscribes on your own” (Custom) beállítás után.

## Beállítások (⚙ a jobb felső sarokban)

- **Import Excel** – ha Excelből dolgozol, a program **csak a beállított nevű fájlt** olvassa be (alapból `Energofish_partner_hirlevel_minta.xlsx`): induláskor a megadott mappából (alapból a program mappája; korábbi változatról frissítve az eddig használt Excel mappája), behúzva vagy tallózva pedig bárhonnan, de csak ezzel a névvel. Más nevű Excelt a program nem tölt be, hanem megmondja, mi a várt név; partner nélküli (rossz) Excelt sem tölt be, a korábbi lista marad. A név és a mappa itt bármikor átírható; az *Adatok* lépésben a **Betöltés** gomb a beállított fájlt tölti be.
- **Cikktörzs (termékfeed)** – a termékkereső forrása (üresen az Energofish nagyker feed).
- **B2B partnertörzs-források** – a célcsoportok tokenes linkjei (titkosítva tárolva, lásd fent).
- **Visszaállítás alapállapotba** – minden beállítás az első indításkori értékre áll (közös tartalom, mintatermékek, sablon, kimeneti mappa, import Excel, cikktörzs címe; kérésre a partnertörzs-források és a mentett partnerhalmazok is). A régi beállításokról másolat készül (`beallitasok-mentes-….json`); a hozzáadott sablonok, a letöltött partnertörzs és a cikktörzs megmaradnak. A jobb felső **⋮** menüből is elérhető – akkor is, ha a felület hibás adat miatt nem töltődik be (ilyenkor a program helyreállító panelt mutat). Végső esetben: `EnergofishHirlevel.exe -alaphelyzet`.
- **Alaphelyzet** – a betöltött **termékek** és/vagy **partnerek** törlése a hírlevélből (a források, a beállítások és a közös tartalom megmaradnak). A *Termékek* lépésben az **Összes törlése** gomb is ezt teszi a termékekkel; a törölt terméklistát a program újraindításkor sem tölti vissza.

## Az Excel felépítése

Tetszőleges oszlopsorrend, a program a **fejléc szövegéből** ismeri fel az oszlopokat (kis-/nagybetű és ékezet nem számít, pl. „E-mail cím”, „Partner e-mail” egyaránt jó).

### 1. munkalap – Partnerek (soronként egy címzett)

| Oszlop | Kötelező | Mire kell |
|---|:-:|---|
| Partner e-mail | ✔ | A címzett. Több cím is lehet pontosvesszővel elválasztva (`bolt@x.hu; tulaj@x.hu`). Hibás/hiányzó cím esetén a sor kimarad. |
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
| Cikk kép link | | fehér hátterű, legalább 260 px-es JPG/PNG (a levélben középre igazítva, kb. 130 px-en látszik; a WEBP-t az Outlook nem jeleníti meg). Üresen: `https://images.energofish.hu/thumbimage/<cikkszám>.JPG` |
| Gomb link | | hová mutasson a gomb és a kép. Üresen: `https://b2b.energofish.hu/termek/<cikkszám>` |
| Rövid leírás | | max. kb. 30 karakter |
| Ár | | számként megadva „3 090 Ft” alakra formázódik |
| Akció | | max. kb. 18 karakter, üres is lehet |
| Aktív | | opcionális: „nem” értéknél a termék nem kerül a levélbe |

A rács asztali nézetben 3 oszlopos (480 px alatti mobilon 2), ezért **3, 6 vagy 9 termék** mutat a legjobban; 1–12 termék bármikor működik.
Linkeknél a cellához rendelt hivatkozás (Ctrl+K) és a `HYPERLINK()` képlet is működik.

### Termék a cikktörzsből (1.2)

A *Termékek* lépés **Új termék** gombja a webshop friss cikktörzsében keres
(`https://energofish.hu/listak/feeds/wholesale/termekadatok_hu.xml`, kb. 12 400 cikk):

- **Keresés cikkszámra vagy névre** – kötőjel nélkül is (`10000327`), ékezet és kis-nagybetű nélkül, több szóval (`wizard crab`). A találatnál látszik a kép, a márka, a kategória, a kisker- és nagyker ár (akcióval), a készlet és a kifutó/kiárusítás jelölés; a már felvett cikkeket a lista jelzi.
- **Hozzáadás** gombbal, dupla kattintással vagy **Enterrel** kerül a termékek közé – egymás után több cikkszám is beírható, az ablak nyitva marad.
- **Kitöltött mezők:** cikkszám, cikknév, rövid leírás (a paraméterekből, pl. „12 cm · Red · Crab”, 30 karakterig), ár, akció („−23%, kifutó”), kép és gomb link. A beszúrás után **minden mező szabadon átírható**.
- **Beállítások az ablakban** (megjegyzi őket): ár – *kisker bruttó* (alap) / *nagyker nettó „+ áfa”* / üresen; előnyben részesített kép – *cikkkép* / nagy / kis / bélyegkép; gomb link – a *webshop termékoldala* vagy a *Haladó beállítások* mintája.
- **Képméret-váltó:** a termékkártyán a cikk összes kitöltött képe (bélyeg-, cikk-, kis, nagy és galériaképek) kis előnézettel és pixelmérettel látszik; egy kattintással cserélhető. Ha az előnyben részesített méret üres, a program a következő kitöltöttet választja. (A *cikkkép* és a *nagy kép* a változat saját képe, a többi a főtermék közös képe.)
- **Excelből jött vagy kézzel felvett termékeknél** is működik: ha a cikkszám szerepel a cikktörzsben, megjelenik a képváltó és a *Kitöltés a cikktörzsből* gomb (az üres mezőket tölti ki).

A cikktörzs a gépre mentődik (`%APPDATA%\EnergofishHirlevel\cikktorzs\`, tömörítve): induláskor onnan azonnal kereshető, az ablak megnyitásakor pedig a program a szerverről csak akkor tölti le újra, ha ott változott. Ha a szerver épp nem érhető el, a legutóbb letöltött változatból keres. A cím az ablak alján (*Cikktörzs címe*) módosítható.

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

Beépítve a tervező három **reszponzív** sablonja (1.1-es változat, lásd [`docs/SABLON-CHANGELOG.md`](docs/SABLON-CHANGELOG.md)): asztali gépen 600 px, tableten középre igazítva, mobilon 2 oszlopos termékrács és egymás alá rendezett gombok. Az előnézetben asztali, tablet (768 px) és mobil (390 px) nézet is választható.

A sablonok a tervezői formátumot követik ([`docs/MEZOK-ES-GENERATOR.md`](docs/MEZOK-ES-GENERATOR.md), `{{kulcs}}` helyőrzők). A program betöltéskor kiegészíti őket; alapesetben a kimenet **bájtra azonos** a tervezői előnézettel (ezt teszt ellenőrzi):

- a fix terméktéglák helyett 1–12 termék (Outlookban is a sablon oszlopszámával);
- **képviselő fotója** kör alakban, narancs kerettel; ha nincs fotó, a monogram;
- a telefon- és e-mail-gomb, a „teljes ajánlat” link, a webes verzió linkje és a közösségi ikonok üres érték esetén elmaradnak;
- kérdés-blokkos sablon (v4): üres kérdésnél a blokk elmarad, a preheaderhez automatikusan hozzáfűzi a „Plusz egy kérdés.” szöveget.

### Új sablon hozzáadása (1.1)

![Sablonválasztó élő bélyegképekkel és a „Sablon hozzáadása” kártyával](docs/kepernyo-sablonok.png)

*Generálás → Sablon → Sablon hozzáadása* (vagy a menüből, vagy egyszerűen az ablakba húzva):

- **egy `.html` sablon** a tervezői formátumban, vagy
- **a tervezőtől kapott `.zip` csomag** – a benne lévő összes sablont felveszi (a kitöltött előnézeteket kihagyja), a saját képeikkel együtt.

A program ellenőrzi a sablont: ismeretlen `{{mezőt}}` használó fájlt nem vesz fel, a hiányzó képeket és a nem átalakítható részeket (pl. nem szabványos termékrács → fix termékhelyek) jelzi. A hozzáadott sablonok a gépen maradnak (`%APPDATA%\EnergofishHirlevel\sablonok\`), átnevezhetők és törölhetők.

**A beépített sablonok frissítése:** ha a tervező a v1/v2/v4 új változatát küldi, ugyanazzal a fájlnévvel (pl. `v4-partnerjelentes.html`) hozzáadva lecseréli a beépítettet – nem kell új program. A frissített változat törlésével az eredeti tér vissza.

> Saját képet (`{{assets.base}}/sajat-kep.png`) használó sablonnál a képet is fel kell tölteni a képtár webcímére. A program a kimenetbe is kimásolja.

---

## Változások

**1.4.1** – a Webgalamb export beolvasása robusztus: lista és számozott kulcsú objektum (lyukakkal, numerikus sorrendben), BOM, burkoló objektum; csak az e-mail cím kötelező, a hiányzó / null / számként érkező mezők nem okoznak hibát; magyar hibaüzenetek sor- és oszlopszámmal; az Ellenőrzés lépésben a beolvasás formátuma és a duplikált címek (melyik rekord maradt ki).

**1.4** – 6. lépés: **Küldés Postmarkon** – négy mód (validálás, sandbox, belső teszt, éles), küldés előtti ellenőrzés (stream-beállítás, letiltott címek, leiratkozottak, képek, méret), kötegelt küldés újrapróbálással, levelenkénti hibakezelés, CSV-napló a dupla küldés ellen, visszajelzések exportja; titkosított tokenek.

**1.3.18** – a cseh célcsoport térképe újra csak Csehország (a helyes forrással a cseh kerületek jönnek).

**1.3.17** – a spanyol („KÖZÖSSÉG - TARTOMÁNY”) Megye-értékek is a térképre kerülnek; a B2B COM célcsoportnál forgó földgömb ráközelítéssel.

**1.3.16** – országtérkép a partnerválasztóban: kattintással szűrhető megyék, a szűrés kiemelve a térképen és a betöltött halmaz kártyáján, mind a 9 célcsoporthoz.

**1.3** – partnerek a B2B partnertörzsből: célcsoport (ország) választás, összetett partnerhalmaz (képviselő, besorolás, partnerbolt, megye, bizományos, belső címek, feliratkozás dátuma, keresés, egyenkénti kizárás, „kivéve”), mentett halmazok, helyi adatbázis szinkronnal (új / frissített / leiratkozott → inaktív, biztonsági zár), frissítés minden generálás előtt, a partnerek saját leiratkozó linkje a láblécben, képviselő-fotók monogramonként, titkosított tokenek.

**1.2** – „Új termék” a friss cikktörzsből: keresés cikkszámra vagy névre, a mezők kitöltése (ár, leírás, akció, kép, link), képméret-váltó a termékkártyán, kitöltés cikkszám alapján; a cikktörzs helyi tárolása és feltételes frissítése.

**1.1** – reszponzív sablonok (mobil + tablet); új sablonok hozzáadása és mentése (.html vagy .zip), a beépítettek frissítése fájlból; tablet előnézet; élő bélyegképek a sablonválasztóban; több e-mail cím egy partnercellában.

**1.0** – első kiadás.

---

## Hibaelhárítás

| Jelenség | Megoldás |
|---|---|
| SmartScreen figyelmeztetés | *További információ → Futtatás mindenképp.* |
| A partnertörzs nem tölthető be / „ismeretlen szerkezetű” JSON | A Webgalamb export lehet lista vagy számozott objektum, az app mindkettőt kezeli. Ha mégis hibát jelez, a hibaüzenet megmondja a fájlnevet és a hiba sorát, oszlopát (pl. csonka letöltés, HTML hibaoldal érvénytelen token miatt). |
| Küldés: „érvénytelen vagy hiányzó API token” | A *Postmark beállítások*ban a szerver *Server API token*jét add meg (nem az Account tokent). |
| Küldés: „a fiók még teszt módban van…” | A Postmark fiók jóváhagyásáig csak `@energofish.hu` címekre lehet küldeni (Belső teszt). |
| Éles küldés tiltva: leiratkozás-kezelés „Postmark” | A Postmarkban a broadcast streamen kapcsold be a „Manage unsubscribes on your own” beállítást (a fiók jóváhagyása után). |
| „bizonytalan” levelek a küldési naplóban | A kapcsolat a küldés közben szakadt meg; a program ezeket nem küldi újra. Nézd meg a Postmark *Activity* oldalán, hogy kimentek-e. |
| „Nem sikerült létrehozni az adatkönyvtárat” (Microsoft Edge), értelmetlen nevű mappák a program mellett, csak rendszergazdaként indul | A korábbi változatok hibája volt (a WebView2 adatmappa útvonala sérülhetett). Töltsd le a legfrissebb kiadást, a program mellett keletkezett furcsa nevű mappákat pedig nyugodtan töröld. Rendszergazdai jog nem kell. Ha mégis előjönne, a `naplo.txt` „WebView2 … betöltő … adatmappa” sora segít a hiba azonosításában. |
| Böngészőben nyílik meg ablak helyett | Hiányzik a WebView2: [telepíthető a Microsofttól](https://developer.microsoft.com/microsoft-edge/webview2/), de böngészőben is minden működik. Kilépés: jobb felső menü → *Kilépés a programból*. |
| Az Excel módosítása nem látszik | Mentsd a fájlt Excelben, majd *Újratöltés*. |
| „A régi .xls formátum nem támogatott” | Excelben *Mentés másként → Excel-munkafüzet (.xlsx)*. |
| A termékképek nem látszanak | Az *Ellenőrzés → Képek ellenőrzése* megmutatja, melyik link rossz. |
| Üres, „lefagyott” felület egy nagy Excel (pl. teljes cikklista) megnyitása után | Az 1.3.14-től a program 50-nél több soros terméklistát nem vesz át (figyelmeztet), legfeljebb 20 000 partnert olvas be, egy levélbe legfeljebb 48 termék kerülhet, és a felület a nagy listákat is gyorsan megjeleníti. Ha a régi változat már elmentette a hibás állapotot: *Termékek › Összes törlése*, vagy *⋮ › Visszaállítás alapállapotba…*. |
| Üres, szétesett felület (pl. rossz Excel megnyitása után) | Jobb felső **⋮** menü → *Visszaállítás alapállapotba…*, vagy a megjelenő helyreállító panel gombjai. Ha a program el sem indul: `EnergofishHirlevel.exe -alaphelyzet` (a régi beállítások másolatként megmaradnak). |
| „Gyanúsan kevés partner” a partnertörzs frissítésekor | Az export a jelenlegi aktívak 80%-ánál kevesebbet adott, ezért a program nem inaktivált senkit. Ha tényleg ennyien maradtak, a figyelmeztetésben kényszeríthető. |
| „A szerver válasza HTTP 403” a partnertörzsnél | A célcsoport tokenje érvénytelen vagy lejárt: *Források…* → új token. |
| „A cikktörzs nem érhető el” | Internetkapcsolat vagy tűzfal/proxy; a hibaüzenet megmondja az okát. Ha korábban már letöltődött, a mentett változatból lehet keresni; terméket kézzel is fel lehet venni. |

Beállítások és napló: `%APPDATA%\EnergofishHirlevel\` (`beallitasok.json`, `naplo.txt`) – a program menüjéből is megnyitható.

---

## Parancssor (haladóknak)

```
EnergofishHirlevel.exe -batch -excel partnerek.xlsx -sablon v4 -kimenet D:\Hirlevel [-tartalom tartalom.json] [-eml] [-felado "Energofish <hirlevel@energofish.hu>"]
```

Felület nélkül generál (pl. ütemezett feladatból); a közös tartalom a mentett beállításokból vagy a `-tartalom` JSON-ból jön, az eredmény a naplóba kerül. `-bongeszo`: saját ablak helyett böngészőben nyílik meg. `-feedteszt [-feed <cím>]`: letölti és feldolgozza a cikktörzset, statisztikát és mintákat ír (a CI is ezzel ellenőrzi az élő feedet). `-partnerteszt B2B_HU`: a partnertörzs élő exportjának ellenőrzése (a token csak a `WEBGALAMB_TOKEN_B2B_HU` környezeti változóból jöhet; a kimenetben csak darabszámok vannak). A CI is lefuttatja, ha a repóban be van állítva a `WEBGALAMB_TOKEN_B2B_HU` titok (*Settings › Secrets and variables › Actions*).

---

## Fejlesztőknek

Go 1.25+, külső futtatókörnyezet nélkül; a felület beágyazott HTML/CSS/JS (Open Sans betűkészlettel), Windowson WebView2 ablakban.

```
./build.sh 1.3.0        # tesztek + dist/EnergofishHirlevel.exe (Linuxról is fordítható)
go test ./...           # egység- és API-tesztek
go run . -bongeszo      # fejlesztői futtatás böngészőben
```

| Hely | Tartalom |
|---|---|
| `main.go`, `app.go` | indítás, helyi HTTP API (csak 127.0.0.1, tokennel védve) |
| `platform_windows.go` | WebView2 ablak, natív fájlválasztók, DPI-kezelés |
| `internal/hirlevel/` | sablonmotor és -átalakító, sablontár, Excel-olvasó, cikktörzs (feed) feldolgozó és keresés, B2B partnertörzs (szinkron, szűrés), validálás, generálás (HTML, EML, CSV) |
| `web/` | a felület |
| `sablonok/`, `assets/` | a tervező nyers (reszponzív) sablonjai és képeik; átalakítás betöltéskor |
| `demo/`, `tools/minta_excel.py` | minta Excel és előállító szkriptje |
| `web/terkepek.json`, `tools/terkepek.py` | a partnerválasztó országtérképei és a földgömb adatai, előállító szkriptjük (Natural Earth közkincs adatokból, mapshaperrel egyszerűsítve; a párosítás: `internal/hirlevel/maps.go`) |
| `web/vendor/` | d3-geo és d3-array (ISC-licenc, `LICENSE-d3.txt`) a földgömb vetítéséhez |
| `app_postmark.go`, `internal/hirlevel/postmark.go`, `send.go` | Postmark-küldés: beállítások és API, a Postmark REST kliens (újrapróbálás, hibakódok), a küldési terv (UTM, leiratkozó fejlécek, kizárások), kötegelés és napló |
| `winres/` | ikon, manifest, verzióinfó (`go-winres make --in winres/winres.json --out rsrc`) |

A `.github/workflows/windows-build.yml` minden pushnál Windows gépen fordít, lefuttatja a teszteket, az öntesztet és egy valódi WebView2-ablakos tesztet, az exe-t pedig letölthető artefaktként csatolja.
