package simple

import (
	"github.com/loadimpact/speedboat/runner"
	"golang.org/x/net/context"
	"net/http"
	"runtime"
	"time"
)

type SimpleRunner struct {
	URL    string
	Client *http.Client
}

func New() *SimpleRunner {
	return &SimpleRunner{
		Client: &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
	}
}

func (r *SimpleRunner) Run(ctx context.Context) <-chan runner.Result {
	ch := make(chan runner.Result)

	go func() {
		defer close(ch)

		for {
			// Note that we abort if we cannot create a request. This means we're either out of
			// memory, or we have invalid user input, neither of which are recoverable.
			req, err := http.NewRequest("GET", r.URL, nil)
			if err != nil {
				ch <- runner.Result{Error: err}
				return
			}
			req.Close = true
			req.Cancel = ctx.Done()

			mem := runtime.MemStats{}
			runtime.ReadMemStats(&mem)
			gcTotal1 := mem.PauseTotalNs

			startTime := time.Now()
			res, err := r.Client.Do(req)
			duration := time.Since(startTime)

			runtime.ReadMemStats(&mem)
			gcTotal2 := mem.PauseTotalNs

			duration -= time.Duration(gcTotal2-gcTotal1) * time.Nanosecond

			select {
			case <-ctx.Done():
				return
			default:
				if err != nil {
					ch <- runner.Result{Error: err, Time: duration}
					continue
				}
				res.Body.Close()
				ch <- runner.Result{Time: duration}
			}
		}
	}()

	return ch
}
