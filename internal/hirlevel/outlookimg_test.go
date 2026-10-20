package hirlevel

import (
	"bytes"
	"image/png"
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

func TestWebpTwinAndHTML(t *testing.T) {
	if n, ok := WebpTwin("ikon/zso.WEBP"); !ok || n != "ikon/zso.png" {
		t.Error(n)
	}
	if _, ok := WebpTwin("x.png"); ok {
		t.Error("png")
	}
	base := "https://energofish.hu/hirlevel/assets"
	for in, want := range map[string]string{
		base + "/zso.webp":                      base + "/zso.png",
		base + "/ikon/level.webp?v=2":           base + "/ikon/level.png?v=2",
		base + "/x.png":                         base + "/x.png",
		"https://images.energofish.hu/k/a.webp": "https://images.energofish.hu/k/a.webp", // más szerver
	} {
		if got := OutlookSafeURL(in, base+"/"); got != want {
			t.Errorf("%s → %s", in, got)
		}
	}
	if OutlookSafeURL("../assets/zso.webp", "") != "../assets/zso.webp" {
		t.Error("helyi képtár")
	}
	html := `<img src="` + base + `/zso.webp" alt=""><td style="background-image:url('` + base + `/osz.WebP');"><img src="https://cdn.example.com/a.webp"><a href="` + base + `/doc.webp.html">`
	got := OutlookSafeHTML(html, base)
	if !strings.Contains(got, base+`/zso.png"`) || !strings.Contains(got, base+`/osz.png')`) || !strings.Contains(got, "cdn.example.com/a.webp") ||
		!strings.Contains(got, "doc.webp.html") {
		t.Error(got)
	}
	if OutlookSafeHTML(html, "") != html {
		t.Error("képtár nélkül")
	}
}

func TestWebpToPNGAndTwins(t *testing.T) {
	photo, _ := os.ReadFile("testdata/foto-minta.webp")
	alpha, _ := os.ReadFile("testdata/level-atlatszo.webp")
	out, err := WebpToPNG(photo)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil || img.Bounds().Dx() != 240 || img.Bounds().Dy() != 200 {
		t.Fatalf("%v %v", err, img.Bounds())
	}
	out, _ = WebpToPNG(alpha)
	img, _ = png.Decode(bytes.NewReader(out))
	if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
		t.Error("az átlátszóság elveszett")
	}
	if _, err := WebpToPNG([]byte("nem kép")); err == nil {
		t.Error("hibás WEBP")
	}
	// a képtár WEBP fájljai mellé PNG pár kerül; meglévő PNG-t nem ír felül, a hibásról figyelmeztet
	files, _ := CollectAssets(fstest.MapFS{"zso.webp": {Data: photo}, "lap.webp": {Data: alpha}, "lap.png": {Data: []byte("sajat")},
		"rossz.webp": {Data: []byte("x")}, "logo.png": {Data: []byte("p")}}, nil, "")
	all, warns := AddWebpTwins(files)
	var names []string
	for _, f := range all {
		names = append(names, f.Rel)
	}
	if strings.Join(names, ",") != "lap.png,lap.webp,logo.png,rossz.webp,zso.webp,zso.png" || len(warns) != 1 || !strings.Contains(warns[0], "rossz.webp") {
		t.Errorf("%v %v", names, warns)
	}
	twin := all[len(all)-1]
	if b, _ := twin.read(); int64(len(b)) != twin.Size || !strings.Contains(twin.Source, "PNG pár") {
		t.Error("pár")
	}
	if CheckRepPhoto(photo) != nil || CheckRepPhoto(out) == nil || CheckRepPhoto(nil) == nil {
		t.Error("fotó-ellenőrzés")
	}
}

func TestSendPlanOutlookImages(t *testing.T) {
	in := sendFixture(t)
	for i := range in.Partners {
		in.Partners[i].RepPhoto = "{assets}/zso.webp"
	}
	plan := BuildSendPlan(in)
	if len(plan.Errors) > 0 {
		t.Fatal(plan.Errors)
	}
	body := plan.Items[0].Msg.HtmlBody
	if !strings.Contains(body, `src="https://kepek.example.com/hirlevel/zso.png"`) || strings.Contains(body, "zso.webp") {
		t.Error("a levélben nem a PNG pár szerepel")
	}
	in.Content[KeepWebpKey] = "1"
	if body := BuildSendPlan(in).Items[0].Msg.HtmlBody; !strings.Contains(body, "zso.webp") || strings.Contains(body, "zso.png") {
		t.Error("WEBP megtartva")
	}
}
