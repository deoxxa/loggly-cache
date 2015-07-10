// command loggly-cache buffers individual log requests to loggly and submits
// them in batches, either 5MB or covering a timespan specified by the user.
package main // import "fknsrs.biz/p/loggly-cache"

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"
)

var (
	addr       = flag.String("addr", ":3000", "Address to listen on.")
	logglyHost = flag.String("loggly_host", "logs-01.loggly.com", "Host to submit logs to.")
	apiKey     = flag.String("api_key", "", "Loggly API key.")
	timeout    = flag.Duration("timeout", time.Second*5, "Maximum time to hold a batch for.")
)

func main() {
	flag.Parse()

	b := batcher{
		host:    *logglyHost,
		key:     *apiKey,
		batches: make(chan []byte, 10),
	}

	go b.run()

	m := http.NewServeMux()

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		before := time.Now()

		fmt.Printf("time=%q message=\"request\"\n", before.Format(time.RFC3339Nano))

		var v interface{}
		if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
			http.Error(w, err.Error(), http.StatusNotAcceptable)
			return
		}

		d, err := json.Marshal(v)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		b.push(d)

		after := time.Now()
		dur := after.Sub(before)

		fmt.Printf("time=%q message=\"response\" took=%s took_ms=%d\n", after.Format(time.RFC3339Nano), dur, dur)
	})

	if err := http.ListenAndServe(*addr, m); err != nil {
		panic(err)
	}
}
