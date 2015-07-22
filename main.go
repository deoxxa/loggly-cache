// command loggly-cache buffers individual log requests to loggly and submits
// them in batches, either of a certain maximum size, or covering a timespan
// specified by the user.
package main // import "fknsrs.biz/p/loggly-cache"

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app        = kingpin.New("loggly-cache", "Cache loggly requests.")
	addr       = app.Flag("addr", "Address to listen on.").Default(":3000").OverrideDefaultFromEnvar("ADDR").String()
	logglyHost = app.Flag("loggly_host", "Host to submit logs to.").Default("logs-01.loggly.com").OverrideDefaultFromEnvar("LOGGLY_HOST").String()
	apiKey     = app.Flag("api_key", "Loggly API key.").OverrideDefaultFromEnvar("API_KEY").String()
	timeout    = app.Flag("timeout", "Maximum time to hold a batch for.").Default("5s").OverrideDefaultFromEnvar("TIMEOUT").Duration()
	batchSize  = kpSizerValue(app.Flag("batch_size", "Maximum size for a batch to get to.").Default("4MB").OverrideDefaultFromEnvar("BATCH_SIZE"))
	quiet      = app.Flag("quiet", "Don't output any request logs.").OverrideDefaultFromEnvar("QUIET").Bool()
)

type logger bool

func (l logger) Printf(tpl string, args ...interface{}) {
	if l {
		fmt.Printf(tpl, args...)
	}
}

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	l := logger(true)
	if *quiet {
		l = false
	}

	b := batcher{
		host:    *logglyHost,
		key:     *apiKey,
		batches: make(chan []byte, 10),
		maxTime: *timeout,
		maxSize: *batchSize,
	}

	go b.run()

	m := http.NewServeMux()

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		before := time.Now()

		l.Printf("time=%q message=\"request\"\n", before.Format(time.RFC3339Nano))

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

		l.Printf("time=%q message=\"response\" took=%s took_ms=%d\n", after.Format(time.RFC3339Nano), dur, dur)
	})

	if err := http.ListenAndServe(*addr, m); err != nil {
		panic(err)
	}
}
