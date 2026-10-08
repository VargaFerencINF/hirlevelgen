package hirlevel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Postmark REST API kliens (csak net/http és encoding/json). A token kizárólag az
// X-Postmark-Server-Token fejlécben utazik: nem kerül címbe, naplóba vagy hibaüzenetbe.

// PostmarkAPI a Postmark API címe.
const PostmarkAPI = "https://api.postmarkapp.com"

// PostmarkTestToken a Postmark „validáló” tokenje: a kérést ellenőrzi, de semmit nem küld el.
const PostmarkTestToken = "POSTMARK_API_TEST"

// PostmarkBatchMax egy kötegben küldhető levelek száma.
const PostmarkBatchMax = 500

// postmarkBatchBytes egy köteg legnagyobb mérete (a Postmark határa 50 MB).
const postmarkBatchBytes = 40 << 20

// PostmarkHeader egy egyedi levélfejléc.
type PostmarkHeader struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

// PostmarkMessage egy levél a /email/batch kéréshez.
type PostmarkMessage struct {
	From          string            `json:"From"`
	To            string            `json:"To"`
	ReplyTo       string            `json:"ReplyTo,omitempty"`
	Subject       string            `json:"Subject"`
	HtmlBody      string            `json:"HtmlBody"`
	TextBody      string            `json:"TextBody,omitempty"`
	MessageStream string            `json:"MessageStream"`
	Tag           string            `json:"Tag,omitempty"`
	Headers       []PostmarkHeader  `json:"Headers,omitempty"`
	Metadata      map[string]string `json:"Metadata,omitempty"`
	TrackOpens    bool              `json:"TrackOpens"`
	TrackLinks    string            `json:"TrackLinks,omitempty"`
}

// PostmarkResult egy levél eredménye a köteg válaszában (HTTP 200 mellett is lehet hibás!).
type PostmarkResult struct {
	ErrorCode   int    `json:"ErrorCode"`
	Message     string `json:"Message"`
	MessageID   string `json:"MessageID"`
	To          string `json:"To"`
	SubmittedAt string `json:"SubmittedAt"`
}

// PostmarkStream egy üzenetfolyam (message stream) adatai.
type PostmarkStream struct {
	ID                                  string `json:"ID"`
	Name                                string `json:"Name"`
	MessageStreamType                   string `json:"MessageStreamType"`
	SubscriptionManagementConfiguration struct {
		UnsubscribeHandlingType string `json:"UnsubscribeHandlingType"`
	} `json:"SubscriptionManagementConfiguration"`
}

// Handling a leiratkozás kezelése: „Custom” (mi kezeljük), „Postmark” vagy „None”.
func (s *PostmarkStream) Handling() string {
	return s.SubscriptionManagementConfiguration.UnsubscribeHandlingType
}

// PostmarkSuppression egy letiltott címzett.
type PostmarkSuppression struct {
	EmailAddress      string `json:"EmailAddress"`
	SuppressionReason string `json:"SuppressionReason"` // HardBounce, SpamComplaint, ManualSuppression
	Origin            string `json:"Origin"`
	CreatedAt         string `json:"CreatedAt"`
}

// SuppressionReasonHU a letiltás okának magyar neve.
func SuppressionReasonHU(r string) string {
	switch r {
	case "HardBounce":
		return "végleges visszapattanás (HardBounce)"
	case "SpamComplaint":
		return "spamnek jelölte (SpamComplaint)"
	case "ManualSuppression":
		return "leiratkozott / letiltva a Postmarkban (ManualSuppression)"
	}
	return r
}

// ErrPostmarkUncertain: a kérés elment, de a válasz nem érkezett meg – nem tudni, kiment-e a levél.
var ErrPostmarkUncertain = errors.New("a Postmark-kérés eredménye ismeretlen (a kapcsolat a küldés közben szakadt meg)")

// PostmarkError a Postmark által elutasított kérés.
type PostmarkError struct {
	Status  int
	Code    int
	Message string
}

func (e *PostmarkError) Error() string {
	if e.Code != 0 || e.Message != "" {
		return fmt.Sprintf("Postmark: %s (HTTP %d)", PostmarkErrorHU(e.Code, e.Message), e.Status)
	}
	return fmt.Sprintf("Postmark: HTTP %d", e.Status)
}

// PostmarkErrorHU a Postmark hibakód érthető magyar leírása (az eredeti üzenettel).
func PostmarkErrorHU(code int, msg string) string {
	hu := map[int]string{
		10:  "érvénytelen vagy hiányzó API token",
		300: "érvénytelen levél (pl. hibás címzett vagy hiányzó mező)",
		400: "a feladó (From) nincs jóváhagyva a Postmarkban",
		401: "a feladó aláírása (Sender Signature) még nincs megerősítve",
		405: "a fiók nem küldhet (elfogyott a kredit vagy felfüggesztették)",
		406: "a címzett le van tiltva (korábbi visszapattanás, spamjelzés vagy leiratkozás)",
		409: "hibás kérés (JSON)",
		410: "túl sok levél egy kötegben",
		412: "a fiók még teszt módban van: csak @energofish.hu címekre lehet küldeni",
	}[code]
	msg = MaskSecrets(strings.TrimSpace(msg))
	switch {
	case hu != "" && msg != "":
		return hu + " – " + msg
	case hu != "":
		return hu
	case code != 0:
		return fmt.Sprintf("hibakód %d: %s", code, msg)
	}
	return msg
}

// PostmarkClient egy szerver-tokennel dolgozó kliens.
type PostmarkClient struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
	Retries int           // 429 és 5xx esetén ennyiszer próbálja újra
	Backoff time.Duration // az első várakozás (utána duplázódik)
	Sleep   func(ctx context.Context, d time.Duration) error
}

// NewPostmarkClient új kliens a megadott szerver-tokennel.
func NewPostmarkClient(token string) *PostmarkClient {
	return &PostmarkClient{BaseURL: PostmarkAPI, Token: token, HTTP: &http.Client{Timeout: 120 * time.Second}, Retries: 5, Backoff: time.Second,
		Sleep: func(ctx context.Context, d time.Duration) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(d):
				return nil
			}
		}}
}

// do egy kérés újrapróbálkozással (429, 5xx, illetve ha a kapcsolat létre sem jött).
// Egy elküldött, de válasz nélkül maradt POST-ot nem ismétel meg: ErrPostmarkUncertain.
func (c *PostmarkClient) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	wait := c.Backoff
	var last error
	for attempt := 0; ; attempt++ {
		var rd io.Reader
		if body != nil {
			rd = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+path, rd)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Postmark-Server-Token", c.Token)
		req.Header.Set("User-Agent", "Energofish-Hirlevel-Generator")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := c.HTTP.Do(req)
		retry := false
		var retryAfter time.Duration
		if err != nil {
			if ctx.Err() != nil {
				if method == http.MethodPost && !notSent(err) {
					return nil, ErrPostmarkUncertain
				}
				return nil, ctx.Err()
			}
			if method == http.MethodPost && !notSent(err) {
				return nil, ErrPostmarkUncertain
			}
			last = fmt.Errorf("a Postmark nem érhető el: %s", SafeNetErr(err))
			retry = true
		} else {
			data, rerr := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
			resp.Body.Close()
			switch {
			case rerr != nil && method == http.MethodPost:
				return nil, ErrPostmarkUncertain
			case rerr != nil:
				last, retry = fmt.Errorf("a Postmark válasza megszakadt: %s", SafeNetErr(rerr)), true
			case resp.StatusCode == http.StatusOK:
				return data, nil
			case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
				last, retry = parsePostmarkError(resp.StatusCode, data), true
				if s, e := strconv.Atoi(resp.Header.Get("Retry-After")); e == nil && s > 0 && s < 120 {
					retryAfter = time.Duration(s) * time.Second
				}
			default:
				return nil, parsePostmarkError(resp.StatusCode, data)
			}
		}
		if !retry || attempt >= c.Retries {
			var pe *PostmarkError
			if method == http.MethodPost && errors.As(last, &pe) && pe.Status >= 500 {
				// a szerverhiba után nem tudni biztosan, mi ment ki
				return nil, fmt.Errorf("%w: %v", ErrPostmarkUncertain, last)
			}
			return nil, last
		}
		d := wait
		if retryAfter > d {
			d = retryAfter
		}
		if err := c.Sleep(ctx, d); err != nil {
			return nil, err
		}
		wait *= 2
	}
}

// notSent igaz, ha a hiba a kapcsolat felépítésekor történt (a kérés biztosan nem ment el).
func notSent(err error) bool {
	var op *net.OpError
	if errors.As(err, &op) && op.Op == "dial" {
		return true
	}
	var dns *net.DNSError
	return errors.As(err, &dns)
}

func parsePostmarkError(status int, data []byte) error {
	var e struct {
		ErrorCode int    `json:"ErrorCode"`
		Message   string `json:"Message"`
	}
	_ = json.Unmarshal(data, &e)
	return &PostmarkError{Status: status, Code: e.ErrorCode, Message: e.Message}
}

// Stream az üzenetfolyam adatai (pl. a leiratkozás kezelésének módja).
func (c *PostmarkClient) Stream(ctx context.Context, id string) (*PostmarkStream, error) {
	data, err := c.do(ctx, http.MethodGet, "/message-streams/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	var s PostmarkStream
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("a Postmark válasza nem értelmezhető: %v", err)
	}
	return &s, nil
}

// Suppressions az üzenetfolyam letiltott címzettjei.
func (c *PostmarkClient) Suppressions(ctx context.Context, stream string) ([]PostmarkSuppression, error) {
	data, err := c.do(ctx, http.MethodGet, "/message-streams/"+url.PathEscape(stream)+"/suppressions/dump", nil)
	if err != nil {
		return nil, err
	}
	var r struct {
		Suppressions []PostmarkSuppression `json:"Suppressions"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("a Postmark válasza nem értelmezhető: %v", err)
	}
	return r.Suppressions, nil
}

// SendBatch egy köteg (legfeljebb 500 levél) elküldése. A válasz levelenként tartalmazza az
// eredményt; a hívónak minden elem ErrorCode-ját ellenőriznie kell.
func (c *PostmarkClient) SendBatch(ctx context.Context, msgs []PostmarkMessage) ([]PostmarkResult, error) {
	if len(msgs) == 0 {
		return nil, nil
	}
	if len(msgs) > PostmarkBatchMax {
		return nil, fmt.Errorf("egy kötegben legfeljebb %d levél lehet", PostmarkBatchMax)
	}
	body, err := json.Marshal(msgs)
	if err != nil {
		return nil, err
	}
	data, err := c.do(ctx, http.MethodPost, "/email/batch", body)
	if err != nil {
		return nil, err
	}
	var res []PostmarkResult
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("%w: a válasz nem értelmezhető", ErrPostmarkUncertain)
	}
	return res, nil
}
