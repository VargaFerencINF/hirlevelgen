#!/usr/bin/env python3
"""A demó bemeneti Excel (demo/Energofish_partner_hirlevel_minta.xlsx) előállítása.

Futtatás:  python3 tools/minta_excel.py
A kész fájl a programba is bekerül („Minta Excel mentése” gomb).
"""
from pathlib import Path

from openpyxl import Workbook
from openpyxl.comments import Comment
from openpyxl.styles import Alignment, Border, Font, PatternFill, Side
from openpyxl.utils import get_column_letter
from openpyxl.worksheet.datavalidation import DataValidation

OUT = Path(__file__).resolve().parent.parent / "demo" / "Energofish_partner_hirlevel_minta.xlsx"

FONT = "Arial"
DARK = "1A171E"
ORANGE = "F1A32B"
GRAPHITE = "414242"
CREAM = "FDF2E1"
LINE = "E6E6E6"

thin = Side(style="thin", color=LINE)
border = Border(left=thin, right=thin, top=thin, bottom=thin)

REPS = {
    "Nagy Attila": ("{assets}/portre-helyettesito.png", "+36 30 123 4567", "nagy.attila@energofish.hu", "Pest és Nógrád megye"),
    "Kiss Zoltán": ("", "+36 30 234 5678", "kiss.zoltan@energofish.hu", "Győr-Moson-Sopron és Vas megye"),
    "Szabó Eszter": ("{assets}/portre-helyettesito.png", "+36 70 345 6789", "szabo.eszter@energofish.hu", "Hajdú-Bihar és Szabolcs megye"),
    "Horváth Balázs": ("", "+36 20 456 7890", "horvath.balazs@energofish.hu", "Bács-Kiskun és Csongrád megye"),
}

# (e-mail, név, cégnév, képviselő)
PARTNERS = [
    ("kapitany.horgaszbolt@example.com", "Kovács Péter", "Kapitány Horgászbolt, Vác", "Nagy Attila"),
    ("pecas.sarok@example.com", "Tóth Erika", "Pecás Sarok Kft., Gödöllő", "Nagy Attila"),
    ("dunakanyar.horgasz@example.com", "Varga Zsolt", "Dunakanyar Horgászcentrum", "Nagy Attila"),
    ("balogh.horgasz@example.com", "Balogh Tamás", "Balogh Horgász- és Vadászbolt", "Nagy Attila"),
    ("info@csali-pont.example.com", "Farkas Judit", "Csali-Pont, Salgótarján", "Nagy Attila"),
    ("rendeles@feederkucko.example.com", "", "Feeder Kuckó, Budaörs", "Nagy Attila"),
    ("gyor.horgaszcentrum@example.com", "Németh Gábor", "Rába Horgászcentrum, Győr", "Kiss Zoltán"),
    ("sopron.pecabolt@example.com", "Pintér Ágnes", "Soproni Pecabolt", "Kiss Zoltán"),
    ("szombathely.halas@example.com", "Molnár László", "Halas Horgászbolt, Szombathely", "Kiss Zoltán"),
    ("mosonmagyarovar.csali@example.com", "Takács Bence", "Lajta Csali, Mosonmagyaróvár", "Kiss Zoltán"),
    ("kormend.horgasz@example.com", "Simon Katalin", "Rába-parti Horgászbolt, Körmend", "Kiss Zoltán"),
    ("debrecen.pontyos@example.com", "Fekete István", "Pontyos Pont, Debrecen", "Szabó Eszter"),
    ("nyiregyhaza.horgasz@example.com", "Oláh Krisztina", "Nyírségi Horgászcentrum", "Szabó Eszter"),
    ("tiszafured.pecas@example.com", "Lakatos Márk", "Tisza-tavi Pecás, Tiszafüred", "Szabó Eszter"),
    ("hajduszoboszlo.csali@example.com", "Papp Dóra", "Csalizó, Hajdúszoboszló", "Szabó Eszter"),
    ("kisvarda.horgasz@example.com", "Jakab Norbert", "Kisvárdai Horgászbolt", "Szabó Eszter"),
    ("kecskemet.horgasz@example.com", "Mészáros Anikó", "Hírös Horgász, Kecskemét", "Horváth Balázs"),
    ("szeged.tiszai@example.com", "Szűcs Ferenc", "Tiszai Horgászcentrum, Szeged", "Horváth Balázs"),
    ("baja.duna@example.com", "Kocsis Réka", "Duna Horgászbolt, Baja", "Horváth Balázs"),
    ("hodmezovasarhely.pecas@example.com", "Bíró Zoltán", "Mártély Pecás", "Horváth Balázs"),
    ("kiskunhalas.csali@example.com", "Vincze Andrea", "Halasi Csali- és Horgászbolt", "Horváth Balázs"),
]

# (cikkszám, cikknév, kép link, gomb link, rövid leírás, ár, akció)
PRODUCTS = [
    ("T82753-204", "Wizard Vertix Vibrato", "https://images.energofish.hu/thumbimage/T82753-204.JPG",
     "https://b2b.energofish.hu/termek/T82753-204", "Csörgős, süllyedő, 6 színben", 3090, "−15% okt. 31-ig"),
    ("T12049-069", "Carp Expert feeder szett", "https://images.energofish.hu/thumbimage/T12049-069.JPG",
     "https://b2b.energofish.hu/termek/T12049-069", "3,60 m bot, orsó, etetőanyag", 40990, "Utolsó 40 db"),
    ("T98134-092", "Benzar Lake Series", "https://images.energofish.hu/thumbimage/T98134-092.JPG",
     "https://b2b.energofish.hu/termek/T98134-092", "Tavi feeder mix, 800 g", 1790, "Raklapra −8%"),
    ("TSG6K0-600", "Seaguar Dulast PEX8", "https://images.energofish.hu/thumbimage/TSG6K0-600.JPG",
     "https://b2b.energofish.hu/termek/TSG6K0-600", "Nyolcszálas fonott, 150 m", 8990, "3+1 akció"),
    ("TSNVC0-007", "Sonik Lockbox Large", "https://images.energofish.hu/thumbimage/TSNVC0-007.JPG",
     "https://b2b.energofish.hu/termek/TSNVC0-007", "Vízálló, töltött szerelékdoboz", 25790, ""),
    ("TM4340-200", "Mustad Infiltrator", "https://images.energofish.hu/thumbimage/TM4340-200.JPG",
     "https://b2b.energofish.hu/termek/TM4340-200", "Súlyozott offset horog", 1815, "−55%, kifutó"),
]


def header_row(ws, headers, required, notes):
    for i, h in enumerate(headers, 1):
        c = ws.cell(row=1, column=i, value=h)
        req = h in required
        c.font = Font(name=FONT, bold=True, color=DARK if req else "FFFFFF", size=10)
        c.fill = PatternFill("solid", fgColor=ORANGE if req else GRAPHITE)
        c.alignment = Alignment(vertical="center", horizontal="left", wrap_text=True, indent=1)
        c.border = border
        if h in notes:
            c.comment = Comment(notes[h], "Energofish")
            c.comment.width, c.comment.height = 320, 110
    ws.row_dimensions[1].height = 34
    ws.freeze_panes = "A2"
    ws.auto_filter.ref = f"A1:{get_column_letter(len(headers))}{ws.max_row}"


def body_style(ws, ncols, widths, link_cols=()):
    for col, w in enumerate(widths, 1):
        ws.column_dimensions[get_column_letter(col)].width = w
    for row in ws.iter_rows(min_row=2, max_row=ws.max_row, max_col=ncols):
        for c in row:
            c.font = Font(name=FONT, size=10, color="0563C1" if c.column in link_cols and c.value else DARK)
            c.alignment = Alignment(vertical="center")
            c.border = border
    for r in range(2, ws.max_row + 1):
        ws.row_dimensions[r].height = 20


def main():
    wb = Workbook()

    # --- Partnerek ---------------------------------------------------------
    ws = wb.active
    ws.title = "Partnerek"
    headers = ["Partner e-mail", "Partner neve", "Területi képviselő", "Képviselő kép link",
               "Képviselő telefon", "Képviselő e-mail", "Képviselő területe", "Cégnév"]
    ws.append(headers)
    for e, n, ceg, rep in PARTNERS:
        photo, tel, mail, region = REPS[rep]
        ws.append([e, n, rep, photo, tel, mail, region, ceg])
    header_row(ws, headers, required={"Partner e-mail", "Partner neve", "Területi képviselő"}, notes={
        "Partner e-mail": "Kötelező. A címzett e-mail címe. Több cím pontosvesszővel elválasztva is megadható.",
        "Partner neve": "A megszólításhoz: „Kedves {nev}!”. Üresen hagyva „Kedves Partnerünk!” lesz.",
        "Területi képviselő": "A képviselő neve. A monogram ebből készül, ha nincs kép.",
        "Képviselő kép link": "Teljes https:// cím, négyzetes kép (min. 128×128). Üresen hagyva monogram jelenik meg. "
                              "A {assets}/… forma a hírlevél saját képtárára mutat.",
        "Képviselő telefon": "Pl. +36 30 123 4567 vagy 06301234567. Üresen hagyva a telefongomb elmarad.",
        "Képviselő e-mail": "A képviselő e-mail gombja. Üresen hagyva a gomb elmarad.",
        "Képviselő területe": "Pl. „Pest és Nógrád megye”, a név alatt jelenik meg.",
        "Cégnév": "Opcionális. A kimeneti fájlnévben és a küldési listában használható, változója: {ceg}.",
    })
    body_style(ws, len(headers), [36, 20, 20, 34, 18, 30, 32, 34], link_cols={4})

    # --- Termékek -----------------------------------------------------------
    wp = wb.create_sheet("Termékek")
    pheaders = ["Cikkszám", "Cikknév", "Cikk kép link", "Gomb link", "Rövid leírás", "Ár", "Akció"]
    wp.append(pheaders)
    for p in PRODUCTS:
        wp.append(list(p))
    header_row(wp, pheaders, required={"Cikkszám", "Cikknév", "Cikk kép link", "Gomb link"}, notes={
        "Cikkszám": "Kötelező. A téglán a név felett jelenik meg.",
        "Cikknév": "Kötelező. Legfeljebb kb. 26 karakter (2 sor).",
        "Cikk kép link": "Fehér hátterű, négyzetes kép, min. 260×260. Üresen hagyva: images.energofish.hu/thumbimage/<cikkszám>.JPG",
        "Gomb link": "Hová mutasson a Rendelés gomb (és a kép). Üresen hagyva: b2b.energofish.hu/termek/<cikkszám>",
        "Rövid leírás": "Opcionális, legfeljebb kb. 30 karakter (2 sor).",
        "Ár": "Opcionális. Számként megadva a program „3 090 Ft” alakra formázza.",
        "Akció": "Opcionális kiemelés, legfeljebb kb. 18 karakter (1 sor).",
    })
    body_style(wp, len(pheaders), [16, 28, 52, 46, 32, 12, 20], link_cols={3, 4})
    for r in range(2, wp.max_row + 1):
        wp.cell(row=r, column=6).number_format = '#,##0" Ft"'
        wp.cell(row=r, column=6).alignment = Alignment(horizontal="right", vertical="center")

    # --- Útmutató -----------------------------------------------------------
    wu = wb.create_sheet("Útmutató")
    wu.sheet_view.showGridLines = False
    wu.column_dimensions["A"].width = 3
    wu.column_dimensions["B"].width = 28
    wu.column_dimensions["C"].width = 92
    rows = [
        ("title", "Energofish partnerhírlevél – bemeneti munkafüzet"),
        ("text", "Ebből a fájlból készül partnerenként a hírlevél. A közös szövegeket (tárgy, borító, levél, lábléc) a programban kell megadni."),
        ("gap", ""),
        ("h", "Mit kell kitölteni?"),
        ("kv", "Partnerek munkalap", "Soronként egy címzett. Ami partnerenként más: e-mail, név, területi képviselő és elérhetőségei."),
        ("kv", "Termékek munkalap", "Az ajánlat termékei, felülről lefelé ebben a sorrendben kerülnek a levélbe. 3, 6 vagy 9 termék mutat a legjobban."),
        ("kv", "Narancs fejléc", "Kötelező oszlop."),
        ("kv", "Szürke fejléc", "Opcionális oszlop, üresen is hagyható."),
        ("kv", "Megjegyzés a fejlécen", "Vidd az egeret a fejléc fölé: részletes leírás az oszlopról."),
        ("gap", ""),
        ("h", "Tudnivalók"),
        ("kv", "Oszlopsorrend", "Tetszőleges. A program a fejléc szövegéből ismeri fel az oszlopokat, a felismerést az „Adatok” fülön mutatja."),
        ("kv", "Új oszlop", "Bármilyen további oszlop (pl. „Partnerkód”) változóként használható a programban: {partnerkod}."),
        ("kv", "Sablonmező felülírása", "Ha egy oszlop fejléce sablonkulcs (pl. note.body), az adott partnernél felülírja a közös szöveget."),
        ("kv", "Linkek", "Teljes https:// cím kell. Cellához rendelt hivatkozás (Ctrl+K) is működik."),
        ("kv", "Termék kiválasztása", "Opcionális „Aktív” oszlop a Termékek lapon: „nem” értéknél a termék betöltődik, de nem kerül a levélbe."),
        ("kv", "Mentés", "Módosítás után mentsd a fájlt, majd a programban nyomd meg az „Újratöltés” gombot."),
        ("gap", ""),
        ("note", "A demóadatok kitaláltak, a partnerek @example.com címei nem valós postafiókok."),
    ]
    r = 2
    for row in rows:
        kind = row[0]
        if kind == "title":
            c = wu.cell(row=r, column=2, value=row[1])
            c.font = Font(name=FONT, bold=True, size=16, color=DARK)
            wu.row_dimensions[r].height = 30
            for col in range(2, 4):
                wu.cell(row=r + 1, column=col).fill = PatternFill("solid", fgColor=ORANGE)
            wu.row_dimensions[r + 1].height = 5
            r += 2
            continue
        if kind == "text":
            c = wu.cell(row=r, column=2, value=row[1])
            wu.merge_cells(start_row=r, start_column=2, end_row=r, end_column=3)
            c.font = Font(name=FONT, size=10, color=GRAPHITE)
            c.alignment = Alignment(wrap_text=True, vertical="top")
            wu.row_dimensions[r].height = 30
        elif kind == "h":
            c = wu.cell(row=r, column=2, value=row[1].upper())
            c.font = Font(name=FONT, bold=True, size=10, color="9E5F0C")
            wu.row_dimensions[r].height = 22
        elif kind == "kv":
            a = wu.cell(row=r, column=2, value=row[1])
            b = wu.cell(row=r, column=3, value=row[2])
            a.font = Font(name=FONT, bold=True, size=10, color=DARK)
            b.font = Font(name=FONT, size=10, color=DARK)
            b.alignment = Alignment(wrap_text=True, vertical="center")
            a.alignment = Alignment(vertical="center")
            fill = None
            if row[1] == "Narancs fejléc":
                fill = ORANGE
            elif row[1] == "Szürke fejléc":
                fill = GRAPHITE
                a.font = Font(name=FONT, bold=True, size=10, color="FFFFFF")
            if fill:
                a.fill = PatternFill("solid", fgColor=fill)
            for c in (a, b):
                c.border = Border(bottom=thin)
            wu.row_dimensions[r].height = 20
        elif kind == "note":
            c = wu.cell(row=r, column=2, value=row[1])
            wu.merge_cells(start_row=r, start_column=2, end_row=r, end_column=3)
            c.font = Font(name=FONT, italic=True, size=9, color="555555")
            c.fill = PatternFill("solid", fgColor=CREAM)
        r += 1

    # egyszerű adatérvényesítés az Aktív-oszlophoz nincs; e-mail formátum figyelmeztetés
    dv = DataValidation(type="custom", formula1='ISNUMBER(FIND("@",A2))', allow_blank=True,
                        showErrorMessage=True, errorStyle="warning",
                        errorTitle="E-mail cím", error="Ez nem tűnik e-mail címnek.")
    ws.add_data_validation(dv)
    dv.add("A2:A2000")

    wb.active = 0
    OUT.parent.mkdir(exist_ok=True)
    wb.save(OUT)
    print("mentve:", OUT)


if __name__ == "__main__":
    main()
