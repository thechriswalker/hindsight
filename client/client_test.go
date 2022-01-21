package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestEvent(t *testing.T) {
	baseHandler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// make the duration measurable.
		time.Sleep(time.Millisecond * 100)
		rw.Write([]byte(`Hello World!\n`))
	})

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Could not listen: %s", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	wg := &sync.WaitGroup{}
	wg.Add(1)
	recv := map[string]interface{}{}
	var recvError error
	go func() {
		t.Logf("temp server on port %d", port)
		err := http.Serve(listener, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer func() {
				wg.Done()
			}()
			if r.Method != "POST" {
				recvError = fmt.Errorf("API got non-POST request: %s", r.Method)
				return
			}
			if r.URL.Path != "/api/ingest" {
				recvError = fmt.Errorf("API got invalid URI: %s", r.URL.Path)
				return
			}
			defer r.Body.Close()
			payload, err := io.ReadAll(r.Body)
			if err != nil {
				recvError = fmt.Errorf("API Failed to read payload: %s", err)
				return
			}
			err = json.Unmarshal(payload, &recv)
			if err != nil {
				recvError = fmt.Errorf("API Failed to parse payload: %s", err)
				return
			}
			//t.Logf("payload: %#v", recv)
		}))
		if err != nil {
			recvError = err
			wg.Done()
		}
	}()

	client := NewClient(
		WithErrorHandler(func(err error) {
			t.Fatalf("OnError: %s", err)
		}),
		WithEndpoint(fmt.Sprintf("http://127.0.0.1:%d/api/ingest", port)),
	)

	dummy := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "https://foo.bar.invalid/baz/quux?foo=bar", nil)
	req.Header.Set("User-Agent", "Some Test Agent")

	Middleware(client)(baseHandler).ServeHTTP(dummy, req)
	// wait for it to finish
	wg.Wait()

	if recvError != nil {
		t.Fatal(recvError)
	}
	if int(recv["BytesWritten"].(float64)) != 14 {
		t.Logf("expceted 'BytesWritten' to be 14, got %v", recv["BytesWritten"])
		t.Fail()
	}
	if recv["UserAgent"].(string) != "Some Test Agent" {
		t.Logf("expceted 'UserAgent' to be 'Some Test Agent', got %q", recv["UserAgent"].(string))
		t.Fail()
	}
	// should be close enough to round to 100
	if int(recv["DurationMS"].(float64)) != 100 {
		t.Logf("expceted 'DurationMS' to be 100, got %v", recv["DurationMS"])
		t.Fail()
	}
}
