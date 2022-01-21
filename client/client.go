package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/askeladdk/httpsyhook"
)

const (
	ClientVersion         = "1.0.0"
	HindsightEventVersion = "1.0"
)

var userAgent string

func init() {
	// build useragent
	userAgent = fmt.Sprintf(
		"Hindsight-Go-Client/%s Go/%s",
		ClientVersion,
		runtime.Version(),
	)
}

type Client struct {
	endpoint   string
	token      string
	trustProxy bool
	onError    func(err error)
	httpClient *http.Client
}

type Option func(c *Client)

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

func WithErrorHandler(fn func(err error)) Option {
	return func(c *Client) {
		c.onError = fn
	}
}

func WithTrustProxy(trust bool) Option {
	return func(c *Client) {
		c.trustProxy = trust
	}
}

func WithEndpoint(url string) Option {
	return func(c *Client) {
		c.endpoint = url
	}
}

func WithApiToken(token string) Option {
	return func(c *Client) {
		if token == "" {
			c.token = ""
		} else {
			// pre-create the authorisation header
			c.token = "Bearer " + token
		}
	}
}

type Event struct {
	httpsyhook.Struct // this is for the dummy methods

	Time         time.Time
	IP           string
	Host         string
	Method       string
	Path         string
	UserAgent    string
	StatusCode   int           // must be a valid code
	BytesWritten int           // must be non-negative
	Duration     time.Duration `json:"-"`
}

func (ev *Event) MarshalJSON() ([]byte, error) {
	type JEvent Event
	return json.Marshal(&struct {
		*JEvent
		// add these 2 fields
		Hindsight  string
		DurationMS int
		// override this field to string
		Time string
	}{
		Hindsight:  HindsightEventVersion,
		DurationMS: int(ev.Duration / time.Millisecond),
		JEvent:     (*JEvent)(ev),
		Time:       ev.Time.UTC().Format(time.RFC3339),
	})
}

// httpsyhook.Interface methods to record stuff
func (ev *Event) HookWriteHeader(w http.ResponseWriter, statusCode int) {
	ev.StatusCode = statusCode
	w.WriteHeader(statusCode)
}
func (ev *Event) HookWrite(w io.Writer, p []byte) (n int, err error) {
	n, err = w.Write(p)
	ev.BytesWritten += n
	return
}

func Middleware(c *Client) func(http.Handler) http.Handler {
	getRemoteIP := func(r *http.Request) string {
		h, _, _ := net.SplitHostPort(r.RemoteAddr)
		if c.trustProxy {
			// read xff header
			xff := strings.Split(r.Header.Get("x-forwarded-for"), ",")
			if len(xff) > 0 {
				h = strings.TrimSpace(xff[0])
			}
		}
		ip := net.ParseIP(h)
		if ip != nil {
			return ip.String()
		}
		return "0.0.0.0"
	}
	return func(h http.Handler) http.Handler {
		// get context, mark start time, add "done" handler to submit event
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			start := time.Now()
			ev := &Event{
				Time:      start,
				IP:        getRemoteIP(req),
				Host:      req.Host,
				Method:    req.Method,
				Path:      req.URL.RequestURI(),
				UserAgent: req.Header.Get("User-Agent"),
				// the status and byteswritten are written by the hooks
			}
			// need to wrap the response writer for the "set header"
			// call..
			h.ServeHTTP(httpsyhook.Wrap(rw, ev), req)
			ev.Duration = time.Since(start)
			// now we must send it to be recorded.
			// we can do this async.
			go c.Record(ev)
		})
	}

}

func NewClient(opts ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) Record(evts ...*Event) {
	sendEvents(c, evts)
}

func sendEvents(c *Client, evts []*Event) {
	if len(evts) == 0 {
		return
	}
	// create a pipe, so we can give the read end to
	// the http request, and write the payloads into
	// the write end
	r, w := io.Pipe()
	// we need to start the request AND then serialise the payloads...
	// so we spawn a goroutine to do the encoding
	go func() {
		enc := json.NewEncoder(w)
		for _, ev := range evts {
			// write the line, discard errors here.
			_ = enc.Encode(ev)
			// add a newline
			n, _ := w.Write([]byte{'\n'})
			if n != 1 {
				// just bail
				return
			}
		}
		w.Close()
	}()
	req, err := http.NewRequest("POST", c.endpoint, r)
	// set the user-agent and headers
	req.Header.Set("user-agent", userAgent)
	if len(evts) == 1 {
		req.Header.Set("content-type", "application/json")
	} else {
		req.Header.Set("content-type", "application/x-ndjson")
	}
	if c.token != "" {
		req.Header.Set("authorisation", c.token)
	}

	if err != nil {
		c.onError(fmt.Errorf("hindsight-sendEvent fail: %w", err))
		return
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		c.onError(fmt.Errorf("hindsight-sendEvent fail: %w", err))
		return
	}
	if res.StatusCode == http.StatusNoContent {
		// this is what we expect
		res.Body.Close()
		return
	}
	// otherwise we should error
	// probably we should read the body, as it will have info.
	// let's read at most 1k from the body, that should be enough for
	// and error message.
	body, err := io.ReadAll(io.LimitReader(res.Body, 1024))
	if err != nil {
		// @TODO...
		c.onError(fmt.Errorf("hindsight-sendEvent err: api response status %d", res.StatusCode))
		return
	}
	// assume the body is utf8
	c.onError(fmt.Errorf("hindsight-sendEvent err: api response(%d): %s", res.StatusCode, string(body)))
}
