package client

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/askeladdk/httpsyhook"
)

const (
	ClientVersion         = "1.0.0"
	HindsightEventVersion = "1.0"
)

type Client struct {
	endpoint   string
	trustProxy bool
	onError    func(err error)
	serializer chan *Event
}

type Option func(c *Client)

func WithEndpoint(addr string) Option {
	return func(c *Client) {
		c.endpoint = addr
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

func getRemoteIP(r *http.Request, trustProxy bool) string {
	h, _, _ := net.SplitHostPort(r.RemoteAddr)
	if trustProxy {
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

func (ev *Event) Finished() {
	ev.Duration = time.Since(ev.Time)
}

func Middleware(c *Client) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		// get context, mark start time, add "done" handler to submit event
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			wrapped, ev := c.Wrap(rw, req)
			h.ServeHTTP(wrapped, req)
			ev.Finished()
			// now we must send it to be recorded.
			// we can do this async.
			go c.Record(ev)

		})
	}
}

func SetRequestValues(req *http.Request, ev *Event, trustProxy bool) {
	ev.IP = getRemoteIP(req, trustProxy)
	ev.Host = req.Host
	ev.Method = req.Method
	ev.Path = req.URL.RequestURI()
	ev.UserAgent = req.Header.Get("User-Agent")
}

func (c *Client) Wrap(rw http.ResponseWriter, req *http.Request) (http.ResponseWriter, *Event) {
	ev := &Event{Time: time.Now()}
	SetRequestValues(req, ev, c.trustProxy)
	return httpsyhook.Wrap(rw, ev), ev
}

type NetJsonEncoder struct {
	endpoint string
	conn     net.Conn
	enc      *json.Encoder
}

func (nj *NetJsonEncoder) connect() (err error) {
	if nj.conn != nil {
		nj.conn.Close()
		nj.conn = nil
	}
	for attempt := 0; attempt < 3; attempt++ {
		nj.conn, err = net.DialTimeout("tcp", nj.endpoint, 100*time.Millisecond)
		if err == nil {
			nj.enc = json.NewEncoder(nj.conn)
			return nil
		}
	}
	return err
}

func (nj *NetJsonEncoder) Encode(v interface{}) error {
	if nj.conn == nil {
		if err := nj.connect(); err != nil {
			return err
		}
	}
	// should not take long
	deadline := 5 * time.Millisecond
	for {
		nj.conn.SetWriteDeadline(time.Now().Add(deadline))
		if err := nj.enc.Encode(v); err != nil {
			// error writing.
			if _, ok := err.(*json.MarshalerError); ok {
				// marshaling error, reconnecting is not going to fix that...
				return err
			}
			// not a marshaling error, likely a "write" error, we should
			// kill the connection and try again.
			if err = nj.connect(); err != nil {
				return err
			}
		} else {
			// success
			return nil
		}
	}
}

func NewClient(opts ...Option) *Client {
	c := &Client{
		endpoint: "127.0.0.1:8765",
		onError:  func(err error) {},
	}
	for _, opt := range opts {
		opt(c)
	}
	c.serializer = make(chan *Event)
	go func() {
		nje := &NetJsonEncoder{endpoint: c.endpoint}
		for ev := range c.serializer {
			err := nje.Encode(ev)
			if err != nil {
				c.onError(err)
			}
		}
	}()
	return c
}

func (c *Client) Record(evts ...*Event) {
	for _, ev := range evts {
		c.serializer <- ev
	}
}
