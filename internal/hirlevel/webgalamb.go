package hirlevel

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// A Webgalamb célcsoport-export gyökéreleme nem állandó: tömb ([{…},{…}]), vagy – ha a PHP-oldal
// egy rekordot kiszűr a lista közepéről, és nem számozza újra – számozott kulcsú objektum
// ({"0":{…},"1":{…},"3":{…}}), akár lyukakkal. Ez a fájl a központi, toleráns beolvasó: minden
// beolvasási pont (letöltés, fájlból betöltés) ezt használja a ParseB2BExport-on keresztül.

// Webgalamb exportformátumok.
const (
	WGFormatList     = "lista"
	WGFormatNumbered = "számozott objektum"
)

// WebgalambData a beolvasott rekordok, eredeti sorrendben.
type WebgalambData struct {
	Records []map[string]any
	Labels  []string // a rekord helye a fájlban: „12.” (lista) vagy „"228" kulcs” (objektum)
	Format  string   // lista / számozott objektum
	Wrapper string   // ha a rekordok egy burkoló objektum kulcsa alatt voltak (pl. „data”)
	Gaps    int      // hiányzó sorszámok a számozott objektumban
	Skipped []string // nem objektum elemek (kimaradnak)
}

// FormatLabel a formátum olvasható leírása.
func (d *WebgalambData) FormatLabel() string {
	s := d.Format
	if d.Gaps > 0 {
		s += fmt.Sprintf(" (%d hiányzó sorszámmal)", d.Gaps)
	}
	if d.Wrapper != "" {
		s += fmt.Sprintf(", a(z) „%s” kulcs alatt", d.Wrapper)
	}
	return s
}

// WebgalambError a beolvasás olvasható, magyar hibája (nyers Go-hibaüzenet nélkül).
type WebgalambError struct {
	Msg       string
	Line, Col int // 0, ha nem ismert
}

func (e *WebgalambError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s (%d. sor, %d. oszlop)", e.Msg, e.Line, e.Col)
	}
	return e.Msg
}

const errUnknownRoot = "A JSON gyökéreleme ismeretlen szerkezetű (várt: lista vagy számozott objektum)"

// DecodeWebgalamb a Webgalamb export rekordjai. Elfogadja a tömböt, a számozott kulcsú
// objektumot (numerikus sorrendben, a lyukakat átugorva), a fájl eleji BOM-ot és a
// szóközöket, valamint a burkoló objektumot ({"data":[…]}, {"rows":{"0":…}}), amelyet a
// rekordok szerkezetéből ismer fel. Üres listára / objektumra 0 rekordot ad, hibát nem.
func DecodeWebgalamb(data []byte) (*WebgalambData, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	trim := bytes.TrimSpace(data)
	if len(trim) == 0 {
		return nil, &WebgalambError{Msg: "A fájl üres (nincs benne JSON)"}
	}
	switch trim[0] {
	case '[', '{':
	case '<':
		return nil, &WebgalambError{Msg: errUnknownRoot + " – a válasz HTML oldal, nem JSON (lehet, hogy a token érvénytelen vagy lejárt)"}
	default:
		return nil, &WebgalambError{Msg: errUnknownRoot}
	}
	dec := json.NewDecoder(bytes.NewReader(trim))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		return nil, syntaxError(trim, err)
	}
	if _, err := dec.Token(); err != io.EOF {
		line, col := position(trim, int(dec.InputOffset()))
		return nil, &WebgalambError{Msg: "A JSON után további, értelmezhetetlen adat következik", Line: line, Col: col}
	}
	out := &WebgalambData{}
	if ok := out.take(root); ok {
		return out, nil
	}
	// burkoló objektum: a rekordokat tartalmazó kulcs keresése (legfeljebb két szint mélyen)
	if obj, ok := root.(map[string]any); ok {
		if key, val := findRecords(obj, 2); key != "" {
			out.Wrapper = key
			out.take(val)
			return out, nil
		}
		if msg, ok := obj["error"].(string); ok || obj["message"] != nil {
			if !ok {
				msg = jsonString(obj["message"])
			}
			return nil, &WebgalambError{Msg: errUnknownRoot + " – a szerver hibát adott: " + MaskSecrets(clip(msg, 160))}
		}
	}
	return nil, &WebgalambError{Msg: errUnknownRoot}
}

// take a gyökér (vagy a burkolt érték) rekordjai, ha lista vagy számozott objektum.
func (d *WebgalambData) take(v any) bool {
	switch x := v.(type) {
	case []any:
		d.Format = WGFormatList
		for i, e := range x {
			if m, ok := e.(map[string]any); ok {
				d.Records = append(d.Records, m)
				d.Labels = append(d.Labels, strconv.Itoa(i+1)+".")
			} else {
				d.Skipped = append(d.Skipped, fmt.Sprintf("a lista %d. eleme nem rekord (%s) – kihagyva", i+1, jsonKind(e)))
			}
		}
		return true
	case map[string]any:
		keys, ok := numberedKeys(x)
		if !ok {
			return false
		}
		d.Format = WGFormatNumbered
		for i, k := range keys {
			if i > 0 {
				d.Gaps += k.n - keys[i-1].n - 1
			}
			m, isObj := x[k.key].(map[string]any)
			if !isObj {
				d.Skipped = append(d.Skipped, fmt.Sprintf("a(z) \"%s\" kulcs értéke nem rekord (%s) – kihagyva", k.key, jsonKind(x[k.key])))
				continue
			}
			d.Records = append(d.Records, m)
			d.Labels = append(d.Labels, fmt.Sprintf("%d. (kulcs: \"%s\")", len(d.Records), k.key))
		}
		return true
	}
	return false
}

type numKey struct {
	n   int
	key string
}

// numberedKeys a számozott objektum kulcsai numerikus sorrendben („2” a „10” előtt, nem
// szövegként rendezve). Üres objektumra üres listát ad; ha egy kulcs nem nemnegatív egész
// szám, az objektum nem számozott.
func numberedKeys(m map[string]any) ([]numKey, bool) {
	keys := make([]numKey, 0, len(m))
	for k := range m {
		n, err := strconv.Atoi(strings.TrimSpace(k))
		if err != nil || n < 0 {
			return nil, false
		}
		keys = append(keys, numKey{n, k})
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].n < keys[j].n })
	return keys, true
}

// recordLike igaz, ha az érték rekordok listája vagy számozott objektuma (legalább egy rekorddal).
func recordLike(v any) (int, bool) {
	switch x := v.(type) {
	case []any:
		n := 0
		for _, e := range x {
			if _, ok := e.(map[string]any); ok {
				n++
			}
		}
		return n, n > 0 && n*2 >= len(x)
	case map[string]any:
		if len(x) == 0 {
			return 0, false
		}
		if _, ok := numberedKeys(x); !ok {
			return 0, false
		}
		n := 0
		for _, e := range x {
			if _, ok := e.(map[string]any); ok {
				n++
			}
		}
		return n, n > 0
	}
	return 0, false
}

// findRecords a burkoló objektumban a rekordokat tartalmazó kulcs (a legtöbb rekordot adó).
func findRecords(obj map[string]any, depth int) (string, any) {
	bestKey, bestN := "", 0
	var best any
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := obj[k]
		if n, ok := recordLike(v); ok && n > bestN {
			bestKey, bestN, best = k, n, v
			continue
		}
		if sub, ok := v.(map[string]any); ok && depth > 1 {
			if sk, sv := findRecords(sub, depth-1); sk != "" {
				if n, _ := recordLike(sv); n > bestN {
					bestKey, bestN, best = k+"."+sk, n, sv
				}
			}
		}
	}
	return bestKey, best
}

func jsonKind(v any) string {
	switch v.(type) {
	case nil:
		return "üres érték"
	case string:
		return "szöveg"
	case json.Number:
		return "szám"
	case bool:
		return "logikai érték"
	case []any:
		return "lista"
	}
	return "objektum"
}

// syntaxError a JSON-hiba magyarul, a hely megadásával.
func syntaxError(data []byte, err error) error {
	var se *json.SyntaxError
	if errors.As(err, &se) {
		line, col := position(data, int(se.Offset)-1) // az eltolás a hibás karakter utánra mutat
		return &WebgalambError{Msg: "A fájl nem érvényes JSON (szintaktikai hiba)", Line: line, Col: col}
	}
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		line, col := position(data, len(data))
		return &WebgalambError{Msg: "A JSON fájl csonka (váratlanul véget ér – a letöltés megszakadhatott)", Line: line, Col: col}
	}
	return &WebgalambError{Msg: "A fájl nem érvényes JSON"}
}

// position a bájt-eltolás sora és oszlopa (1-től számozva, karakterekben).
func position(data []byte, off int) (int, int) {
	if off > len(data) {
		off = len(data)
	}
	if off < 0 {
		off = 0
	}
	before := data[:off]
	line := bytes.Count(before, []byte("\n")) + 1
	last := bytes.LastIndexByte(before, '\n')
	return line, utf8.RuneCount(before[last+1:]) + 1
}
