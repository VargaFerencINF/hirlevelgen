package hirlevel

import (
	"strings"
	"testing"
)

const fakeToken = "0123456789abcdef0123456789abcdef" // kitalált

func TestAutoLoginLinks(t *testing.T) {
	for _, tc := range []struct{ u, want string }{
		{"https://energofish.hu/product.php/x/24021/", "https://energofish.hu/product.php/x/24021/?v=" + fakeToken},
		{"https://b2b.energofish.hu/offer.php?id=273", "https://b2b.energofish.hu/offer.php?id=273&v=" + fakeToken},
		{"https://energofish.hu/hot-sale?utm_source=a#top", "https://energofish.hu/hot-sale?utm_source=a&v=" + fakeToken + "#top"},
		{"https://www.energofish.cz/x", "https://www.energofish.cz/x?v=" + fakeToken},
		{"https://partnerbolt.hu/energofish.hu", "https://partnerbolt.hu/energofish.hu"}, // külső domain
		{"https://energofish.hu.rossz.com/x", "https://energofish.hu.rossz.com/x"},
		{"mailto:info@energofish.hu", "mailto:info@energofish.hu"},
	} {
		if got := AddLogin(tc.u, "v="+fakeToken); got != tc.want {
			t.Errorf("%s → %s", tc.u, got)
		}
	}
	if AddLogin("https://energofish.hu/x", "") != "https://energofish.hu/x" {
		t.Error("token nélkül")
	}
	// a küldéskori kampány-UTM a token elé kerül, a v= marad a legvégén
	if got := AppendKeepLogin("https://energofish.hu/x?id=1&v="+fakeToken+"#a", "utm_source=pb&utm_medium=email"); got != "https://energofish.hu/x?id=1&utm_source=pb&utm_medium=email&v="+fakeToken+"#a" {
		t.Error(got)
	}
	if got := AppendKeepLogin("https://energofish.hu/x/?v="+fakeToken, "utm_source=pb"); got != "https://energofish.hu/x/?utm_source=pb&v="+fakeToken {
		t.Error(got)
	}
	if got := AppendKeepLogin("https://energofish.hu/x", "utm_source=pb"); got != "https://energofish.hu/x?utm_source=pb" {
		t.Error(got)
	}
	if LoginParam(&Partner{Token: "rossz token"}) != "" || LoginParam(nil) != "" || LoginParam(&Partner{Token: fakeToken}) != "v="+fakeToken {
		t.Error("LoginParam")
	}
}

func TestBuildAutoLogin(t *testing.T) {
	c := Content{"offer.urlPattern": "https://b2b.energofish.hu/termek/{cikkszam}", "offer.more.url": "https://energofish.hu/akciok",
		"offer.linkParams": "utm_source=hirlevel&utm_campaign=okt"}
	prods := []Product{{On: true, Code: "10000-327", URL: "https://energofish.hu/product.php/x/24021/"}, {On: true, Code: "A-1"},
		{On: true, Code: "K-1", URL: "https://kulso.example.com/termek"}}
	p := &Partner{Email: "a@example.com", Token: fakeToken}
	// kikapcsolva: változatlan
	d := Build(c, prods, p, nil, "")
	if strings.Contains(d.Items[0]["url"], "v=") || strings.Contains(d.Values["offer.more.url"], "v=") {
		t.Errorf("kikapcsolva: %v", d.Items[0]["url"])
	}
	c[AutoLoginKey] = "1"
	d = Build(c, prods, p, nil, "")
	if d.Items[0]["url"] != "https://energofish.hu/product.php/x/24021/?utm_source=hirlevel&utm_campaign=okt&v="+fakeToken ||
		d.Items[1]["url"] != "https://b2b.energofish.hu/termek/A-1?utm_source=hirlevel&utm_campaign=okt&v="+fakeToken ||
		d.Items[2]["url"] != "https://kulso.example.com/termek?utm_source=hirlevel&utm_campaign=okt" ||
		d.Values["offer.more.url"] != "https://energofish.hu/akciok?utm_source=hirlevel&utm_campaign=okt&v="+fakeToken {
		t.Errorf("bekapcsolva: %v %v", d.Items, d.Values["offer.more.url"])
	}
	// token nélküli partner (pl. Excelből): nincs paraméter
	d = Build(c, prods, &Partner{Email: "b@example.com"}, nil, "")
	if strings.Contains(d.Items[0]["url"], "v=") {
		t.Error("token nélkül")
	}
	// a leiratkozó link sosem kap tokent
	d = Build(c, prods, &Partner{Email: "c@example.com", Token: fakeToken, Unsubscribe: "https://energofish.hu/leiratkozas.html?&c=0123456789abcdef0123"}, nil, "")
	if strings.Contains(d.Values["footer.unsubscribe.url"], "v=") {
		t.Error("leiratkozó link")
	}
	// a kapcsoló a tartalom normalizálása után is megmarad
	if n := c.Normalize(Content{}); n[AutoLoginKey] != "1" {
		t.Error("normalizálás")
	}
}

func TestSendPlanAutoLogin(t *testing.T) {
	in := sendFixture(t)
	in.Partners[0].Token = fakeToken
	in.Content[AutoLoginKey] = "1"
	plan := BuildSendPlan(in)
	if len(plan.Errors) > 0 {
		t.Fatal(plan.Errors)
	}
	body := plan.Items[0].Msg.HtmlBody
	n := strings.Count(body, "v="+fakeToken)
	// a token a link legvégén, a kampány-UTM után áll
	if n == 0 || strings.Count(body, "&amp;v="+fakeToken+`"`)+strings.Count(body, "?v="+fakeToken+`"`) != n ||
		strings.Count(body, "utm_campaign=2026-10+%C5%91szi&amp;v="+fakeToken) != n {
		t.Errorf("token a linkekben: %d", n)
	}
	if !strings.Contains(plan.Items[0].Msg.TextBody, "&v="+fakeToken) {
		t.Error("szöveges változat")
	}
	// a leiratkozó linkre nem kerül
	if strings.Contains(body, "leiratkozas.html?&amp;c=00000000000000000001&amp;v=") {
		t.Error("leiratkozó link tokennel")
	}
	// a többi partner levelébe nem kerül token
	if strings.Contains(plan.Items[1].Msg.HtmlBody, fakeToken) {
		t.Error("más partner levele")
	}
	// teszt módokban a valódi token nem kerül a levélbe, helyette a helyettesítő
	for _, mode := range []string{ModeValidate, ModeSandbox, ModeInternal} {
		in.Mode = mode
		p := BuildSendPlan(in)
		for _, it := range p.Items {
			if strings.Contains(it.Msg.HtmlBody, fakeToken) || strings.Contains(it.Msg.TextBody, fakeToken) {
				t.Fatalf("%s: valódi token teszt módban", mode)
			}
		}
		if len(p.Items) == 0 || (mode != ModeInternal && !strings.Contains(p.Items[0].Msg.HtmlBody, "v="+TestLoginToken)) {
			t.Errorf("%s: nincs helyettesítő", mode)
		}
	}
}
