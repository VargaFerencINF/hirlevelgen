package hirlevel

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRepPhotoFromAssets(t *testing.T) {
	for in, want := range map[string]string{"ZSO": "zso.png", " GÁL ": "gal.png", "ŐRY": "ory.png", "SZ AN": "szan.png", "": "", "/": ""} {
		if got := RepPhotoFile(in); got != want {
			t.Errorf("%q → %q", in, got)
		}
	}
	// alapból a képtárból jön, a kézzel megadott fotó felülírja
	o := B2BMapOptions{RepPhotos: map[string]string{"HIM": "https://kep.example.com/him.jpg"}}
	a := B2BToPartner(&B2BPartner{Email: "a@example.com", RepName: "Gál János", RepMono: "GÁL"}, 1, "B2B_HU", o)
	b := B2BToPartner(&B2BPartner{Email: "b@example.com", RepName: "Halmos Mihály", RepMono: "HIM"}, 2, "B2B_HU", o)
	c := B2BToPartner(&B2BPartner{Email: "c@example.com"}, 3, "B2B_HU", o)
	if a.RepPhoto != "{assets}/gal.png" || b.RepPhoto != "https://kep.example.com/him.jpg" || c.RepPhoto != "" {
		t.Errorf("%q %q %q", a.RepPhoto, b.RepPhoto, c.RepPhoto)
	}
	content := Content{"assets.base": "https://energofish.hu/hirlevel/b2b/assets/"}
	if d := Build(content, nil, &a, nil, ""); d.Values["rep.photo"] != "https://energofish.hu/hirlevel/b2b/assets/gal.png" {
		t.Errorf("levélben: %q", d.Values["rep.photo"])
	}
}

func TestRepPhotoCheck(t *testing.T) {
	var buf bytes.Buffer
	_ = png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 200, 200)))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/assets/zso.png" {
			_, _ = w.Write(buf.Bytes())
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	content := Content{"assets.base": srv.URL + "/assets"}
	o := B2BMapOptions{}
	partners := []Partner{
		B2BToPartner(&B2BPartner{Email: "a@example.com", RepName: "Szabó Zsolt", RepMono: "ZSO"}, 1, "B2B_HU", o),
		B2BToPartner(&B2BPartner{Email: "b@example.com", RepName: "Szabó Zsolt", RepMono: "ZSO"}, 2, "B2B_HU", o),
		B2BToPartner(&B2BPartner{Email: "c@example.com", RepName: "Gál János", RepMono: "GÁL"}, 3, "B2B_HU", o),
	}
	var reps []ImageTarget
	for _, tg := range CollectImageTargets(content, nil, partners, nil) {
		if tg.Kind == "rep" {
			reps = append(reps, tg)
		}
	}
	// képviselőnként egyszer, a fájlnévvel (<monogram>.png)
	if len(reps) != 2 || reps[0].Where != "Képviselő fotója: Szabó Zsolt (zso.png)" || reps[1].Where != "Képviselő fotója: Gál János (gal.png)" {
		t.Fatalf("%+v", reps)
	}
	res := CheckImages(context.Background(), reps)
	if res[0].Status == "error" || res[1].Status != "error" || !strings.Contains(res[1].Message, "404") {
		t.Errorf("%+v", res)
	}
	// a „WEBP megtartása” kapcsoló a PNG fotót nem érinti
	content[KeepWebpKey] = "1"
	for _, tg := range CollectImageTargets(content, nil, partners[:1], nil) {
		if tg.Kind == "rep" && tg.Where != "Képviselő fotója: Szabó Zsolt (zso.png)" {
			t.Errorf("WEBP megtartva: %+v", tg)
		}
	}
}

// B2B partnernél a képviselő neve alatt nincs megye; Excel-partnernél a megadott terület marad.
func TestRepRegionHiddenForB2B(t *testing.T) {
	b := B2BToPartner(&B2BPartner{Email: "a@example.com", RepName: "Gál János", RepMono: "GÁL", County: "Pest"}, 1, "B2B_HU", B2BMapOptions{})
	d := Build(Content{"note.body": "Üdv {terulet}"}, nil, &b, nil, "")
	if d.Values["rep.region"] != "" || !strings.Contains(d.Values["note.body"], "Pest") {
		t.Errorf("B2B: %q %q", d.Values["rep.region"], d.Values["note.body"])
	}
	e := Partner{Email: "e@example.com", RepName: "Nagy Attila", RepRegion: "Pest és Nógrád megye"}
	if d := Build(Content{}, nil, &e, nil, ""); d.Values["rep.region"] != "Pest és Nógrád megye" {
		t.Errorf("Excel: %q", d.Values["rep.region"])
	}
}

// „A területi képviselő írja alá”: név, beosztás és fotó a képviselőé; képviselő nélkül a közös aláíró.
func TestSignerRep(t *testing.T) {
	c := Content{"assets.base": "https://kep.example.com/a", "note.signer.name": "Tóth Gábor", "note.signer.title": "értékesítési igazgató",
		"note.signer.portrait": "{assets}/toth.png", "note.signer.repTitle": "területi képviselő ({terulet})"}
	b := B2BToPartner(&B2BPartner{Email: "a@example.com", RepName: "Gál János - Energofish Kft.", RepMono: "GÁL", County: "Pest"}, 1, "B2B_HU", B2BMapOptions{})
	none := Partner{Email: "x@example.com"}
	nophoto := Partner{Email: "y@example.com", RepName: "Kiss Anna"}
	if d := Build(c, nil, &b, nil, ""); d.Values["note.signer.name"] != "Tóth Gábor" {
		t.Error("kikapcsolva")
	}
	c[SignerRepKey] = "1"
	d := Build(c, nil, &b, nil, "")
	if d.Values["note.signer.name"] != "Gál János" || d.Values["note.signer.title"] != "területi képviselő (Pest megye)" ||
		d.Values["note.signer.portrait"] != "https://kep.example.com/a/gal.png" {
		t.Errorf("képviselő: %q %q %q", d.Values["note.signer.name"], d.Values["note.signer.title"], d.Values["note.signer.portrait"])
	}
	if d := Build(c, nil, &nophoto, nil, ""); d.Values["note.signer.name"] != "Kiss Anna" || d.Values["note.signer.portrait"] != "https://kep.example.com/a/portre-helyettesito.png" {
		t.Errorf("fotó nélkül: %q", d.Values["note.signer.portrait"])
	}
	if d := Build(c, nil, &none, nil, ""); d.Values["note.signer.name"] != "Tóth Gábor" || d.Values["note.signer.portrait"] != "https://kep.example.com/a/toth.png" {
		t.Errorf("képviselő nélkül: %q", d.Values["note.signer.name"])
	}
}
