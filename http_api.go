package hindsight

import (
	"bufio"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func CreateAPIHandler(c *Config, store Storage) http.Handler {
	r := mux.NewRouter()
	// the api only has a single endpoint, so the mux.Router is
	// a little bit overkill, but we might add more...
	r.Methods("POST").Path("/api/ingest").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("content-type") {
		case "application/x-ndjson":
			// ndjson
			// we need to be a little stricter than the default streaming parser
			// we will line buffer
			sc := bufio.NewScanner(r.Body)
			defer r.Body.Close()
			i := 0
			for sc.Scan() {
				in := &InboundEvent{}
				err := json.Unmarshal(sc.Bytes(), in)
				if err != nil {
					// bail here.
					respondJSON(rw, http.StatusUnprocessableEntity, map[string]interface{}{
						"OK":       false,
						"Status":   http.StatusUnprocessableEntity,
						"Ingested": i,
						"Error":    err.Error(),
					})
					return
				}
				err = store.Store(mapInboundEvent(c, in))
				if err != nil {
					// bail here.
					respondJSON(rw, http.StatusInternalServerError, map[string]interface{}{
						"OK":       false,
						"Status":   http.StatusInternalServerError,
						"Ingested": i,
						"Error":    err.Error(),
					})
					return
				}
				i++
			}
			if err := sc.Err(); err != nil {
				respondJSON(rw, http.StatusBadRequest, map[string]interface{}{
					"OK":       false,
					"Status":   http.StatusBadRequest,
					"Ingested": i,
					"Error":    err.Error(),
				})
				return
			}
			respondJSON(rw, http.StatusOK, map[string]interface{}{
				"OK":       true,
				"Status":   http.StatusOK,
				"Ingested": i,
			})
		case "application/json", "text/json":
			// regular json.
			in := &InboundEvent{}
			defer r.Body.Close()
			err := json.NewDecoder(r.Body).Decode(in)
			if err != nil {
				respondJSON(rw, http.StatusBadRequest, map[string]interface{}{
					"OK":       false,
					"Status":   http.StatusBadRequest,
					"Ingested": 0,
					"Error":    err.Error(),
				})
				return
			}
			err = store.Store(mapInboundEvent(c, in))
			if err != nil {
				// bail here.
				respondJSON(rw, http.StatusInternalServerError, map[string]interface{}{
					"OK":       false,
					"Status":   http.StatusInternalServerError,
					"Ingested": 0,
					"Error":    err.Error(),
				})
				return
			}
			respondJSON(rw, http.StatusOK, map[string]interface{}{
				"OK":       true,
				"Status":   http.StatusOK,
				"Ingested": 1,
			})
		default:
			respondJSON(rw, http.StatusNotAcceptable, map[string]interface{}{
				"OK":     false,
				"Status": http.StatusNotAcceptable,
				"Error":  "Unknown content type",
			})
		}
	})

	return r
}

func respondJSON(rw http.ResponseWriter, status int, data interface{}) {
	rw.Header().Set("content-type", "application/json")
	rw.WriteHeader(status)
	json.NewEncoder(rw).Encode(data)
}
