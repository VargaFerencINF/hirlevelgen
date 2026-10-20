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
	for in, want := range map[string]string{"ZSO": "zso.webp", " GÁL ": "gal.webp", "ŐRY": "ory.webp", "SZ AN": "szan.webp", "": "", "/": ""} {
		if got := RepPhotoFile(in); got != want {
			t.Errorf("%q → %q", in, got)
		}
	}
	// alapból a képtárból jön, a kézzel megadott fotó felülírja
	o := B2BMapOptions{RepPhotos: map[string]string{"HIM": "https://kep.example.com/him.jpg"}}
	a := B2BToPartner(&B2BPartner{Email: "a@example.com", RepName: "Gál János", RepMono: "GÁL"}, 1, "B2B_HU", o)
	b := B2BToPartner(&B2BPartner{Email: "b@example.com", RepName: "Halmos Mihály", RepMono: "HIM"}, 2, "B2B_HU", o)
	c := B2BToPartner(&B2BPartner{Email: "c@example.com"}, 3, "B2B_HU", o)
	if a.RepPhoto != "{assets}/gal.webp" || b.RepPhoto != "https://kep.example.com/him.jpg" || c.RepPhoto != "" {
		t.Errorf("%q %q %q", a.RepPhoto, b.RepPhoto, c.RepPhoto)
	}
	content := Content{"assets.base": "https://energofish.hu/hirlevel/b2b/assets/"}
	if d := Build(content, nil, &a, nil, ""); d.Values["rep.photo"] != "https://energofish.hu/hirlevel/b2b/assets/gal.webp" {
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
	// képviselőnként egyszer, a fájlnévvel; Outlook-barát módban (alap) a PNG párt nézi
	if len(reps) != 2 || reps[0].Where != "Képviselő fotója: Szabó Zsolt (zso.png)" || reps[1].Where != "Képviselő fotója: Gál János (gal.png)" {
		t.Fatalf("%+v", reps)
	}
	res := CheckImages(context.Background(), reps)
	if res[0].Status == "error" || res[1].Status != "error" || !strings.Contains(res[1].Message, "404") {
		t.Errorf("%+v", res)
	}
	// a WEBP megtartásával a .webp fájlt nézi
	content[KeepWebpKey] = "1"
	for _, tg := range CollectImageTargets(content, nil, partners[:1], nil) {
		if tg.Kind == "rep" && tg.Where != "Képviselő fotója: Szabó Zsolt (zso.webp)" {
			t.Errorf("WEBP megtartva: %+v", tg)
		}
	}
}
