#!/usr/bin/env python3
"""A tervező által átadott sablonok (docs/eredeti-sablonok) átalakítása a generátor
saját sablonjaivá (sablonok/). Egyszer kell lefuttatni, ha új sablonverzió érkezik.

Változtatások (alapesetben a kimenet bájtra azonos az eredetivel):
  * a 6 fix terméktégla helyett {{#grid offer.items}} … {{/grid}} blokk (1-12 termék);
  * képviselő: ha van fotó ({{rep.photo}}), kör alakú kép, különben a monogram;
  * képviselő telefon- és e-mail gombja csak akkor jelenik meg, ha van érték;
  * v4: az „Egy kérdés” blokk és a borító „1 kérdés” felirata elhagyható;
  * a böngésző-link, a „teljes ajánlat” link és a közösségi ikonok elhagyhatók.
"""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / "internal" / "hirlevel" / "testdata" / "eredeti"
DST = ROOT / "sablonok"

SEP = re.compile(r'<!--\[if mso\]></td>(?:</tr><tr>)?<td width="177" valign="top"><!\[endif\]-->\n?')


def wrap(html, start, end, cond, label):
    """A start..end (end-et is beleértve) szakaszt {{#if cond}}…{{/if}} közé teszi."""
    i = html.find(start)
    if i < 0:
        sys.exit(f"{label}: nem található kezdet: {start[:60]}")
    j = html.find(end, i)
    if j < 0:
        sys.exit(f"{label}: nem található vég: {end[:60]}")
    j += len(end)
    return html[:i] + "{{#if " + cond + "}}" + html[i:j] + "{{/if}}" + html[j:]


def convert(name):
    html = (SRC / name).read_text(encoding="utf-8")

    # --- terméktégla-rács -------------------------------------------------
    first = html.find('<div class="tile"')
    last_end = html.find("<!--[if mso]></td></tr></table><![endif]-->", first)
    grid = html[first:last_end]
    tail_ws = grid[len(grid.rstrip()):]
    grid = grid.rstrip()
    tiles = SEP.split(grid)
    if len(tiles) != 6:
        sys.exit(f"{name}: {len(tiles)} tégla, 6 volt várható")
    body = tiles[0].replace("offer.items.1.", "item.")
    for n, t in enumerate(tiles, 1):
        if t.replace(f"offer.items.{n}.", "item.") != body:
            sys.exit(f"{name}: a(z) {n}. tégla eltér az elsőtől")
    html = html[:first] + "{{#grid offer.items}}" + body + "{{/grid}}" + tail_ws + html[last_end:]

    # --- képviselő fotó / monogram ------------------------------------------
    m = re.search(r'<td width="64" height="64"[^>]*>\{\{rep\.initials\}\}</td>', html)
    if not m:
        sys.exit(f"{name}: monogram-cella nem található")
    initials_td = m.group(0)
    photo_td = (
        '<td width="70" height="70" valign="middle" style="width:70px;height:70px;">'
        '<img src="{{rep.photo}}" alt="{{rep.name}}" width="64" height="64" '
        'style="display:block;width:64px;height:64px;border:3px solid #F1A32B;border-radius:50%;'
        "background-color:#F1A32B;font-family:'Open Sans',Arial,Helvetica,sans-serif;font-size:12px;"
        'line-height:16px;color:#1A171E;"></td>'
    )
    html = html.replace(initials_td, "{{#if rep.photo}}" + photo_td + "{{else}}" + initials_td + "{{/if}}")

    # --- képviselő gombok ----------------------------------------------------
    html = wrap(html, '<td class="stack" style="padding:0 10px 10px 0;">', "{{rep.phone}}</a></td></tr></table></td>",
                "rep.phone.url", name + " telefon")
    html = wrap(html, '<td class="stack" style="padding:0 0 10px 0;">', "{{rep.email}}</a></td></tr></table></td>",
                "rep.email.url", name + " e-mail")

    # --- elhagyható linkek ----------------------------------------------------
    html = wrap(html, '<a href="{{utility.browserLink.url}}"', "{{utility.browserLink}}</a>",
                "utility.browserLink.url", name + " böngésző-link")
    html = wrap(html, '<a href="{{offer.more.url}}"', "{{offer.more}}</a>", "offer.more.url", name + " teljes ajánlat")
    for net in ("facebook", "youtube", "instagram", "tiktok"):
        m = re.search(r'<td style="padding-right:8px;"><a href="\{\{social\.' + net + r'\.url\}\}">.*?</a></td>', html)
        if not m:
            sys.exit(f"{name}: {net} ikon nem található")
        html = html.replace(m.group(0), "{{#if social." + net + ".url}}" + m.group(0) + "{{/if}}")

    # --- v4: kérdés blokk -----------------------------------------------------
    if "poll.question" in html:
        html = wrap(html, '&nbsp;&nbsp;·&nbsp;&nbsp;<span style="white-space:nowrap;">{{cover.meta.poll}}</span>',
                    "{{cover.meta.poll}}</span>", "poll.question", name + " borító kérdés")
        seg_start = html.rfind("<tr>", 0, html.find("{{poll.label}}"))
        m = re.compile(r"\{\{poll\.note\}\}</div>\s*</td></tr>").search(html, seg_start)
        if seg_start < 0 or not m:
            sys.exit(f"{name}: kérdés blokk határai nem találhatók")
        j = m.end()
        html = html[:seg_start] + "{{#if poll.question}}" + html[seg_start:j] + "{{/if}}" + html[j:]
        for n in (1, 2, 3):
            m = re.search(r'<table role="presentation"[^>]*margin-bottom:10px;"><tr><td[^>]*>'
                          r'<a href="\{\{poll\.answers\.' + str(n) + r'\.url\}\}"[^>]*>\{\{poll\.answers\.' + str(n) +
                          r'\}\}</a></td></tr></table>\n?', html)
            if not m:
                sys.exit(f"{name}: {n}. válasz nem található")
            html = html.replace(m.group(0), "{{#if poll.answers." + str(n) + "}}" + m.group(0) + "{{/if}}")

    (DST / name).write_text(html, encoding="utf-8")
    print(f"{name}: kész ({len(html)} bájt)")


if __name__ == "__main__":
    DST.mkdir(exist_ok=True)
    for n in ("v1-sotet-lemez.html", "v2-waterside.html", "v4-partnerjelentes.html"):
        convert(n)
