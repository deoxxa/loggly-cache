package main

import (
	"bytes"
	"fmt"
	"net/http"
	"time"
)

const maxSize = 1024 * 1024 * 5

type batcher struct {
	host    string
	key     string
	buffer  []byte
	batches chan []byte
}

func (b *batcher) push(d []byte) {
	if len(b.buffer)+len(d)+1 > maxSize {
		b.flush()
	}

	b.buffer = append(b.buffer, append(d, '\n')...)
}

func (b *batcher) flush() {
	if len(b.buffer) > 0 {
		fmt.Printf("time=%q message=flush bytes=%d\n", time.Now().Format(time.RFC3339Nano), len(b.buffer))

		b.batches <- b.buffer
		b.buffer = nil
	}
}

func (b *batcher) run() {
	for {
		select {
		case <-time.After(time.Second * 5):
			b.flush()
		case data := <-b.batches:
			go func(data []byte) {
				before := time.Now()

				fmt.Printf("time=%q message=\"submit batch\" bytes=%d\n", before.Format(time.RFC3339Nano), len(data))

				res, err := http.Post("http://"+b.host+"/bulk/"+b.key, "text/plain", bytes.NewReader(data))
				if err != nil {
					fmt.Printf("time=%q message=\"request error\" error=%q\n", time.Now().Format(time.RFC3339Nano), err.Error())
					b.batches <- data
					return
				}

				if res.StatusCode != 200 {
					fmt.Printf("time=%q message=\"response status\" status=%d\n", time.Now().Format(time.RFC3339Nano), res.StatusCode)
					return
				}

				after := time.Now()
				dur := after.Sub(before)

				fmt.Printf("time=%q message=\"submit batch complete\" took=%s took_ms=%d\n", after.Format(time.RFC3339Nano), dur, dur)
			}(data)
		}
	}
}
