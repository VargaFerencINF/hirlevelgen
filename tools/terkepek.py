#!/usr/bin/env python3
"""A partnerválasztó országtérképeinek előállítása (web/terkepek.json).

Forrás: Natural Earth (közkincs, https://www.naturalearthdata.com/):
  ne_10m_admin_1_states_provinces.geojson (megyék, régiók) és
  ne_50m_admin_0_countries.geojson (országok, a nemzetközi célcsoporthoz).
Eszköz: mapshaper (npm install mapshaper) az egyszerűsítéshez és a vetítéshez.

Használat:  python3 tools/terkepek.py <natural-earth-mappa> <mapshaper> web/terkepek.json
A kimenet régiónként: azonosító, felirat, SVG-útvonal és a névváltozatok (a partnertörzs
Megye mezőjének párosításához; a párosítás a programban, Go-ban történik).
"""
import json, os, subprocess, sys, tempfile

SRC, MAPSHAPER, OUT = sys.argv[1], sys.argv[2], sys.argv[3]

# célcsoport → (ország, a térkép neve, egyszerűsítés)
GROUPS = {
    "B2B_HU": ("HUN", "Magyarország", "interval=900"),
    "B2B_SK": ("SVK", "Szlovákia", "interval=900"),
    "B2B_CZ": ("CZE", "Csehország", "interval=1100"),
    "B2B_AT": ("AUT", "Ausztria", "interval=1100"),
    "B2B_DE": ("DEU", "Németország", "interval=2500"),
    "B2B_RO": ("ROU", "Románia", "interval=1800"),
    "B2B_ES": ("ESP", "Spanyolország", "interval=2500"),
    "B2B_RS": ("SRB", "Szerbia", "interval=1100"),
}

# a megyei jogú városok a Natural Earthben külön egységek: a megyéjükhöz olvasztjuk
HU_CITIES = {
    "HU-BC": "HU-BE", "HU-DE": "HU-HB", "HU-DU": "HU-FE", "HU-EG": "HU-HE", "HU-ED": "HU-PE", "HU-GY": "HU-GS",
    "HU-HV": "HU-CS", "HU-KV": "HU-SO", "HU-KM": "HU-BK", "HU-MI": "HU-BZ", "HU-NK": "HU-ZA", "HU-NY": "HU-SZ",
    "HU-PS": "HU-BA", "HU-ST": "HU-NO", "HU-SN": "HU-GS", "HU-SD": "HU-CS", "HU-SF": "HU-FE", "HU-SS": "HU-TO",
    "HU-SK": "HU-JN", "HU-SH": "HU-VA", "HU-TB": "HU-KE", "HU-VM": "HU-VE", "HU-ZE": "HU-ZA",
}

# régiócsoportok (egy Megye-érték több régiót is jelölhet), a Natural Earth „region” mezőjén túl
EXTRA_GROUPS = {
    "SRB": {"Vojvodina|Vajdaság|Војводина": ["RS-01", "RS-02", "RS-03", "RS-04", "RS-05", "RS-06", "RS-07"]},
}
# további névváltozatok régiónként
EXTRA_KEYS = {
    "HU-CS": ["Csongrád", "Csongrád-Csanád"],
    "HU-BU": ["Budapest", "Bp.", "Főváros"],
    "CZ-PR": ["Praha", "Hlavní město Praha", "Prága"],
    "RS-00": ["Beograd", "Belgrád", "Grad Beograd"],
    "RO-B": ["București", "Bucuresti", "Bukarest", "Municipiul București"],
    "AT-9": ["Wien", "Bécs", "Vienna"],
}
ES_GROUP_KEYS = {  # a spanyol autonóm közösségek további nevei
    "Andalucía": ["Andalusia", "Andalúzia"], "Cataluña": ["Catalunya", "Catalonia", "Katalónia"],
    "País Vasco": ["Euskadi", "Basque Country", "Baszkföld"], "Valenciana": ["Comunidad Valenciana", "Comunitat Valenciana", "Valencia"],
    "Islas Baleares": ["Illes Balears", "Baleares", "Baleár-szigetek"], "Canary Is.": ["Canarias", "Islas Canarias", "Kanári-szigetek"],
    "Castilla y León": ["Kasztília és León"], "Castilla-La Mancha": ["Kasztília-La Mancha"], "Foral de Navarra": ["Navarra"],
    "Madrid": ["Comunidad de Madrid"], "Murcia": ["Región de Murcia"], "Asturias": ["Principado de Asturias"],
}

NAME_FIELDS = ["name", "name_local", "woe_name", "gn_name", "name_en", "name_de", "name_hu", "name_es", "name_fr", "name_it", "name_pl"]


def keys_of(p, fields, alt_field):
    out = []
    for k in fields:
        v = p.get(k)
        if v:
            out.append(str(v))
    for v in str(p.get(alt_field) or "").replace(",", "|").split("|"):
        if v.strip():
            out.append(v.strip())
    seen, res = set(), []
    for v in out:
        if v.lower() not in seen:
            seen.add(v.lower())
            res.append(v)
    return res


def run_mapshaper(features, simplify, lat0, lon0, dissolve=False, clip=None):
    """clip: téglalap a vetített síkon (méter), a vetítés után."""
    with tempfile.TemporaryDirectory() as td:
        src, dst = os.path.join(td, "in.json"), os.path.join(td, "out.json")
        json.dump({"type": "FeatureCollection", "features": features}, open(src, "w"))
        cmd = [MAPSHAPER, src]
        if dissolve:
            cmd += ["-dissolve", "rid"]
        cmd += ["-proj", f"+proj=laea +lat_0={lat0} +lon_0={lon0} +datum=WGS84"]
        if clip:
            cmd += ["-clip", "bbox=" + ",".join(map(str, clip)), "remove-slivers"]
        cmd += [
                "-simplify", simplify, "weighting=0.7", "keep-shapes",
                "-o", dst, "format=geojson", "precision=1"]
        subprocess.run(cmd, check=True, capture_output=True)
        return json.load(open(dst))["features"]


def rings(geom):
    if geom is None:
        return []
    if geom["type"] == "Polygon":
        return geom["coordinates"]
    if geom["type"] == "MultiPolygon":
        return [r for poly in geom["coordinates"] for r in poly]
    return []


def to_svg(features, width=320, max_h=230, shift=None):
    """Méretezés a nézőablakra; a shift a régiók eltolása (pl. Kanári-szigetek) a vetített síkban."""
    xs, ys = [], []
    for f in features:
        dx, dy = (shift or {}).get(f["properties"]["rid"], (0, 0))
        for r in rings(f["geometry"]):
            for x, y in r:
                xs.append(x + dx); ys.append(y + dy)
    minx, maxx, miny, maxy = min(xs), max(xs), min(ys), max(ys)
    s = min((width - 4) / (maxx - minx), (max_h - 4) / (maxy - miny))
    w, h = (maxx - minx) * s + 4, (maxy - miny) * s + 4
    def pt(x, y, dx, dy):
        return (round((x + dx - minx) * s + 2, 1), round((maxy - (y + dy)) * s + 2, 1))
    out = {}
    for f in features:
        rid = f["properties"]["rid"]
        dx, dy = (shift or {}).get(rid, (0, 0))
        parts = []
        for r in rings(f["geometry"]):
            pts, last = [], None
            for x, y in r:
                p = pt(x, y, dx, dy)
                if p != last:
                    pts.append(p); last = p
            if len(pts) < 4:
                continue
            parts.append("M" + " ".join(f"{fmt(a)} {fmt(b)}" for a, b in pts[:-1]) + "Z")
        out[rid] = out.get(rid, "") + "".join(parts)
    return out, f"0 0 {fmt(round(w, 1))} {fmt(round(h, 1))}", (minx, maxy, s)


def fmt(v):
    t = f"{v:.1f}"
    return t[:-2] if t.endswith(".0") else t


def main():
    adm1 = json.load(open(os.path.join(SRC, "ne_10m_admin_1_states_provinces.geojson")))["features"]
    adm0 = json.load(open(os.path.join(SRC, "ne_50m_admin_0_countries.geojson")))["features"]
    result = {"_forras": "Natural Earth (közkincs), egyszerűsítve – tools/terkepek.py", "maps": {}}

    for group, (a3, title, simp) in GROUPS.items():
        feats = [f for f in adm1 if f["properties"]["adm0_a3"] == a3]
        meta, groups, src = {}, {}, []
        for f in feats:
            p = f["properties"]
            rid = p["iso_3166_2"]
            if a3 == "HUN" and rid in HU_CITIES:
                rid = HU_CITIES[rid]
            else:
                label = p["name"]
                if a3 == "HUN":
                    label = p.get("name_hu") or p["name"]
                elif a3 == "SRB":
                    label = (p.get("region") if p.get("region") not in (None, "None") else None) or p["name"]
                    if rid == "RS-19":
                        label = "Rasinski"
                    if rid == "RS-00":
                        label = "Beograd"
                elif a3 == "CZE" and rid == "CZ-PR":
                    label = "Praha"
                keys = keys_of(p, NAME_FIELDS, "name_alt") + EXTRA_KEYS.get(rid, [])
                if a3 == "SRB" and rid == "RS-19":
                    keys = [k for k in keys if "omorav" not in k.lower()] + ["Rasinski", "Rasina", "Raszinai"]
                hu = p.get("name_hu")
                meta[rid] = {"id": rid, "label": label, "hu": hu if hu and hu != label else "", "keys": keys}
                reg = p.get("region")
                if a3 == "ESP" and reg:
                    groups.setdefault(reg, []).append(rid)
            g = dict(f); g["properties"] = {"rid": rid}
            src.append(g)
        extra_group_keys = {}
        for k, ids in EXTRA_GROUPS.get(a3, {}).items():
            first, *rest = k.split("|")
            groups[first] = ids
            extra_group_keys[first] = rest
        lat0 = sum(f["properties"]["latitude"] for f in feats) / len(feats)
        lon0 = sum(f["properties"]["longitude"] for f in feats) / len(feats)
        if a3 == "ESP":
            lat0, lon0 = 40.0, -3.7
        out = run_mapshaper(src, simp, round(lat0, 2), round(lon0, 2), dissolve=(a3 == "HUN"))
        shift = None
        frame = ""
        if a3 == "ESP":  # a Kanári-szigetek kerettel a félsziget alá, balra
            canary = set(groups.get("Canary Is.", []))
            cx = [c[0] for f in out if f["properties"]["rid"] in canary for r in rings(f["geometry"]) for c in r]
            cy = [c[1] for f in out if f["properties"]["rid"] in canary for r in rings(f["geometry"]) for c in r]
            mx = [c[0] for f in out if f["properties"]["rid"] not in canary for r in rings(f["geometry"]) for c in r]
            my = [c[1] for f in out if f["properties"]["rid"] not in canary for r in rings(f["geometry"]) for c in r]
            dx = min(mx) - min(cx) + 10000
            dy = min(my) - max(cy) - 25000  # a félsziget alá
            shift = {rid: (dx, dy) for rid in canary}
        paths, vb, (minx, maxy, s) = to_svg(out, shift=shift)
        if shift:
            dx, dy = next(iter(shift.values()))
            canary = set(shift)
            cx = [c[0] + dx for f in out if f["properties"]["rid"] in canary for r in rings(f["geometry"]) for c in r]
            cy = [c[1] + dy for f in out if f["properties"]["rid"] in canary for r in rings(f["geometry"]) for c in r]
            x0, x1 = (min(cx) - minx) * s + 2 - 4, (max(cx) - minx) * s + 2 + 4
            y0, y1 = (maxy - max(cy)) * s + 2 - 4, (maxy - min(cy)) * s + 2 + 4
            frame = f"M{fmt(round(x0,1))} {fmt(round(y0,1))}H{fmt(round(x1,1))}V{fmt(round(y1,1))}H{fmt(round(x0,1))}Z"
        regions = []
        for rid in sorted(meta, key=lambda r: meta[r]["label"]):
            if rid not in paths or not paths[rid]:
                print("hiányzó geometria:", group, rid, file=sys.stderr)
                continue
            m = meta[rid]; m["d"] = paths[rid]
            regions.append(m)
        gl = []
        for name, ids in sorted(groups.items()):
            if len(ids) < 2 and name not in ES_GROUP_KEYS:
                continue
            gl.append({"keys": [name] + ES_GROUP_KEYS.get(name, []) + extra_group_keys.get(name, []), "ids": sorted(ids)})
        result["maps"][group] = {"title": title, "kind": "megye", "viewBox": vb, "frame": frame, "regions": regions, "groups": gl}

    # nemzetközi célcsoport: Európa országai
    eu = [f for f in adm0 if f["properties"]["CONTINENT"] == "Europe" or f["properties"]["ADM0_A3"] in ("TUR", "CYP")]
    meta, src = {}, []
    for f in eu:
        p = f["properties"]
        rid = p["ISO_A2"] if p["ISO_A2"] not in ("-99", None) else p["ADM0_A3"]
        keys = [p[k] for k in ("NAME", "NAME_LONG", "FORMAL_EN", "NAME_EN", "NAME_HU", "NAME_DE", "NAME_ES", "NAME_FR", "NAME_IT", "NAME_PL", "NAME_SORT", "NAME_CIAWF") if p.get(k)]
        keys += [p["ADM0_A3"]] + ([p["ISO_A2"]] if p["ISO_A2"] not in ("-99", None) else [])
        meta[rid] = {"id": rid, "label": p.get("NAME_HU") or p["NAME"], "hu": "", "keys": list(dict.fromkeys(keys))}
        g = dict(f); g["properties"] = {"rid": rid}
        src.append(g)
    out = run_mapshaper(src, "interval=9000", 52, 12, clip=[-2500000, -2050000, 2350000, 2350000])
    paths, vb, _ = to_svg(out, width=320, max_h=260)
    regions = []
    for rid in sorted(meta, key=lambda r: meta[r]["label"]):
        if paths.get(rid):
            m = meta[rid]; m["d"] = paths[rid]
            regions.append(m)
    result["maps"]["B2B_COM"] = {"title": "Európa", "kind": "ország", "viewBox": vb, "frame": "", "regions": regions, "groups": []}

    json.dump(result, open(OUT, "w", encoding="utf-8"), ensure_ascii=False, separators=(",", ":"))
    for g, m in result["maps"].items():
        print(g, len(m["regions"]), "régió,", sum(len(r["d"]) for r in m["regions"]) // 1024, "KB", m["viewBox"])
    print("összesen", os.path.getsize(OUT) // 1024, "KB")


main()
