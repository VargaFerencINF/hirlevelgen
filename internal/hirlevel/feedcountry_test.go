package hirlevel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFeedCountryFormats(t *testing.T) {
	for in, want := range map[string]float64{"3090": 3090, "12.90": 12.9, "12,90": 12.9, " 1 290,50 ": 1290.5, "1.290.50": 1290.5, "": 0, "x": 0, "-5": 0} {
		if got := parseNum(in); got != want {
			t.Errorf("%q → %v", in, got)
		}
	}
	for _, c := range []struct {
		v       float64
		country string
		cur     string
		want    string
	}{
		{3090, "hu", "", "3 090 Ft"},
		{12.9, "at", "", "12,90 €"},
		{1290, "cz", "", "1 290 Kč"},
		{1290.5, "sk", "", "1 290,50 €"},
		{12.9, "en", "", "€12.90"},
		{1234.5, "en", "", "€1,234.50"},
		{49.9, "ro", "", "49,90 lei"},
		{49.9, "ro", "RON", "49,90 RON"},
	} {
		if got := FormatMoney(c.v, c.country, c.cur); got != c.want {
			t.Errorf("%v %s %s → %q (várt %q)", c.v, c.country, c.cur, got, c.want)
		}
	}
	for g, want := range map[string]string{"B2B_HU": "hu", "B2B_CZ": "cz", "B2B_SK": "sk", "B2B_AT": "at", "B2B_DE": "at", "B2B_COM": "en", "B2B_RS": "en", "B2B_RO": "ro", "B2B_ES": "es", "": "hu"} {
		if got := FeedCountryForGroup(g); got != want {
			t.Errorf("%s → %s", g, got)
		}
	}
	if FeedURLFor("cz") != "https://energofish.hu/listak/feeds/wholesale/termekadatok_cz.xml" || FeedURLFor("hu") != DefaultFeedURL {
		t.Error("cím")
	}
}

// a külföldi cikktörzsek szerkezete: <Product> angol mezőnevekkel, tizedesvesszős (néhol ponttal
// írt, 4 tizedes) árakkal – a valódi termekadatok_cz.xml egy cikke alapján, rövidítve
const czFeed = `<?xml version="1.0" encoding="utf-8"?>
<xml>
<Product>
	<Item_code>10000327</Item_code>
	<Item_codeK>10000-327</Item_codeK>
	<Product_name><![CDATA[WIZARD CRAB 12CM]]></Product_name>
	<Main_product id="24021"><![CDATA[WIZARD CRAB]]></Main_product>
	<Product_link>https://www.energofish.cz/product.php/x/24021/</Product_link>
	<Category id="210"><![CDATA[Nástrahy - Přívlač]]></Category>
	<Subcategory id="210040"><![CDATA[Gumové nástrahy]]></Subcategory>
	<Brand_manufacturer><![CDATA[Arno]]></Brand_manufacturer>
	<Wholesale_net_price>15,50</Wholesale_net_price>
	<Wholesale_discount_net_price>12.4000</Wholesale_discount_net_price>
	<Retail_gross_price>27,90</Retail_gross_price>
	<Retail_discount_gross_price>0,00</Retail_discount_gross_price>
	<On_stock>1</On_stock>
	<Images>
		<Image id="code">https://images.energofish.hu/codeimage/C10000-327.JPG</Image>
	</Images>
	<Parameters>
		<Parameter id="5" name="Kód zboží"><![CDATA[10000-327]]></Parameter>
		<Parameter id="21" name="Velikost"><![CDATA[12 cm]]></Parameter>
		<Parameter id="40" name="Akce"><![CDATA[B]]></Parameter>
	</Parameters>
	<Low_stock>0</Low_stock>
	<Product_sticker>58p_253p_icon_hu.png</Product_sticker>
	<Subproduct_sticker>56s_kifuto_plecsni.png</Subproduct_sticker>
</Product>
</xml>`

func TestCountryFeedStore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(czFeed)) }))
	defer srv.Close()
	st := NewFeedStore(t.TempDir())
	st.Country = "cz"
	st.Refresh(srv.URL, true)
	st.Wait(10 * time.Second)
	list := st.Products()
	if s := st.Status(); s.Country != "cz" || s.Label != "Cseh (CZ)" || len(list) != 1 || list[0].Country != "cz" || list[0].Wholesale != 15.5 || list[0].WholesaleAkc != 12.4 ||
		list[0].Family != "WIZARD CRAB" || list[0].FamilyID != 24021 || list[0].Subcategory != "Gumové nástrahy" || list[0].Stock != 1 || FeedDesc(&list[0], 30) != "12 cm" {
		t.Fatalf("%+v %+v", s, list)
	}
	// nagyker nettó: cseh pénznem és ÁFA-felirat, a kedvezmény csak százalék (a matrica magyar szó)
	p := list[0].ToProduct(FeedOptions{Price: "wholesale"})
	if p.Price != "12,40 Kč bez DPH" || p.Deal != "−20%" || p.URL != "https://www.energofish.cz/product.php/x/24021/" {
		t.Errorf("%q %q %q", p.Price, p.Deal, p.URL)
	}
	if p := list[0].ToProduct(FeedOptions{Currency: map[string]string{"cz": "CZK"}}); p.Price != "27,90 CZK" || p.Deal != "" {
		t.Errorf("kisker: %q %q", p.Price, p.Deal)
	}
	// újraindítás: a gyorsítótárból is cseh cikkek
	st2 := NewFeedStore(st.Dir)
	st2.Country = "cz"
	st2.LoadCached(srv.URL)
	st2.Wait(10 * time.Second)
	if l := st2.Products(); len(l) != 1 || l[0].Country != "cz" || !strings.Contains(FormatMoney(l[0].Retail, "cz", ""), "Kč") {
		t.Error("gyorsítótár")
	}
	// a nemzetközi cikktörzs nem létező energofish.en linkjei az energofish.com-ra javulnak
	en := strings.Replace(czFeed, "https://www.energofish.cz/product.php/x/24021/", "https://energofish.en/product.php/Lures-WIZARD-CRAB/24021/", 1)
	srvEN := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(en)) }))
	defer srvEN.Close()
	st3 := NewFeedStore(t.TempDir())
	st3.Country = "en"
	st3.Refresh(srvEN.URL, true)
	st3.Wait(10 * time.Second)
	if l := st3.Products(); len(l) != 1 || l[0].Link != "https://energofish.com/product.php/Lures-WIZARD-CRAB/24021/" {
		t.Errorf("EN link: %+v", l)
	}
}
