package caddylog

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/0x6377/hindsight/client"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var (
	errBadLog = errors.New("hindsight logger recieved non-access log")
)

func init() {
	caddy.RegisterModule(HindsightEncoder{})
}

type HindsightEncoder struct {
	*RequestObjectEncoder `json:"-"`
	bpool                 buffer.Pool
	TrustProxy            bool `json:"trust_proxy"`
}

func (he HindsightEncoder) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "caddy.logging.encoders.hindsight",
		New: func() caddy.Module {
			return new(HindsightEncoder)
		},
	}
}

func (he *HindsightEncoder) Provision(ctx caddy.Context) error {
	he.RequestObjectEncoder = &RequestObjectEncoder{}
	he.bpool = buffer.NewPool()
	return nil
}

func (he *HindsightEncoder) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if d.NextArg() {
			if he.TrustProxy {
				return fmt.Errorf("too many args")
			}
			var s string
			d.Args(&s)
			if s == "trust_proxy" {
				he.TrustProxy = true
			} else {
				return fmt.Errorf("invalid arg")
			}
		}
	}
	return nil
}

func (he HindsightEncoder) Clone() zapcore.Encoder {
	return HindsightEncoder{
		bpool:                he.bpool,
		RequestObjectEncoder: he.RequestObjectEncoder,
		TrustProxy:           he.TrustProxy,
	}
}

func (he HindsightEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	// we only care about specific fields and if we cannot
	// find them, then we just "don't log"
	// first find the field for "logger" which should be "http.log.access"

	// Now that our logger is a simple one, we can piggyback the zapcore.jsonEncoder
	// with a "specific" config.
	ev := &client.Event{}
	if he.RequestObjectEncoder.Req == nil {
		return nil, errBadLog
	}
	client.SetRequestValues(he.Req, ev, he.TrustProxy)
	required := 0
	for _, f := range fields {
		switch f.Key {
		case "duration":
			ev.Duration = time.Duration(f.Integer)
			ev.Time = time.Now().Add(-1 * ev.Duration)
			required++
		case "size":
			ev.BytesWritten = int(f.Integer)
			required++
		case "status":
			ev.StatusCode = int(f.Integer)
			required++
		}
	}
	if required != 3 {
		// we didn't get the data
		return nil, errBadLog
	}
	// should be a pool!
	buf := he.bpool.Get()
	err := json.NewEncoder(buf).Encode(ev)
	if err != nil {
		buf.Free()
		return nil, err
	}

	return buf, nil
}

type RequestObjectEncoder struct {
	Req *http.Request
}

var _ zapcore.ObjectEncoder = (*RequestObjectEncoder)(nil)

// need to implement the entire object encoder spec.
func (re *RequestObjectEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	return nil
}
func (re *RequestObjectEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	// this is the request!
	if key == "request" {
		if req, ok := marshaler.(caddyhttp.LoggableHTTPRequest); ok {
			re.Req = req.Request
		}
	}
	return nil
}

// Built-in types.
func (re *RequestObjectEncoder) AddBinary(key string, value []byte)          {} // for arbitrary bytes
func (re *RequestObjectEncoder) AddByteString(key string, value []byte)      {} // for UTF-8 encoded bytes
func (re *RequestObjectEncoder) AddBool(key string, value bool)              {}
func (re *RequestObjectEncoder) AddComplex128(key string, value complex128)  {}
func (re *RequestObjectEncoder) AddComplex64(key string, value complex64)    {}
func (re *RequestObjectEncoder) AddDuration(key string, value time.Duration) {}
func (re *RequestObjectEncoder) AddFloat64(key string, value float64)        {}
func (re *RequestObjectEncoder) AddFloat32(key string, value float32)        {}
func (re *RequestObjectEncoder) AddInt(key string, value int)                {}
func (re *RequestObjectEncoder) AddInt64(key string, value int64)            {}
func (re *RequestObjectEncoder) AddInt32(key string, value int32)            {}
func (re *RequestObjectEncoder) AddInt16(key string, value int16)            {}
func (re *RequestObjectEncoder) AddInt8(key string, value int8)              {}
func (re *RequestObjectEncoder) AddString(key, value string)                 {}
func (re *RequestObjectEncoder) AddTime(key string, value time.Time)         {}
func (re *RequestObjectEncoder) AddUint(key string, value uint)              {}
func (re *RequestObjectEncoder) AddUint64(key string, value uint64)          {}
func (re *RequestObjectEncoder) AddUint32(key string, value uint32)          {}
func (re *RequestObjectEncoder) AddUint16(key string, value uint16)          {}
func (re *RequestObjectEncoder) AddUint8(key string, value uint8)            {}
func (re *RequestObjectEncoder) AddUintptr(key string, value uintptr)        {}

// AddReflected uses reflection to serialize arbitrary objects, so it can be
// slow and allocation-heavy.
func (re *RequestObjectEncoder) AddReflected(key string, value interface{}) error { return nil }

// OpenNamespace opens an isolated namespace where all subsequent fields will
// be added. Applications can use namespaces to prevent key collisions when
// injecting loggers into sub-components or third-party libraries.
func (re *RequestObjectEncoder) OpenNamespace(key string) {}
