package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/uber-go/tally"
	"github.com/uber-go/tally/m3"
)

const (
	numCities   = 500
	numDevics   = 100
	numCounters = 10

	concurrency = 100
)

var n = flag.Int("n", 1000, "number of metrics to emit in 1000s")

func main() {
	opts := m3.Options{
		// this field is required, no server is listening on
		// this port, so stats reporting will fail, but that
		// is okay, error is ignored anyway
		HostPorts:          []string{"localhost:8888"},
		Env:                "test",
		Service:            "foo",
		MaxQueueSize:       100000,
		MaxPacketSizeBytes: 1440,
	}
	reporter, err := m3.NewReporter(opts)
	if err != nil {
		log.Fatal(err)
	}

	scope, closer := tally.NewRootScope(
		tally.ScopeOptions{
			CachedReporter: reporter,
			ExpiryPeriod:   time.Duration(2) * time.Second,
		},
		time.Duration(1000)*time.Millisecond,
	)

	cities := make([]string, numCities)
	for i := 0; i < numCities; i++ {
		cities[i] = "city" + strconv.Itoa(i)
	}
	devices := make([]string, numDevics)
	for i := 0; i < numDevics; i++ {
		devices[i] = "version" + strconv.Itoa(i)
	}
	counters := make([]string, numCounters)
	for i := 0; i < numCounters; i++ {
		counters[i] = "counter" + strconv.Itoa(i)
	}

	flag.Parse()
	log.Println("number of ops:", strconv.Itoa(*n)+"k")

	ch := make(chan struct{}, concurrency)
	count := 0
	for count < *n*1000 {
		ch <- struct{}{}
		go func() {
			tags := map[string]string{
				"city":   cities[rand.Intn(numCities)],
				"device": devices[rand.Intn(numDevics)],
			}
			// assume it takes 100ms to do work
			time.Sleep(time.Duration(100) * time.Microsecond)
			scope.Tagged(tags).Counter(counters[rand.Intn(numCounters)]).Inc(1)
			<-ch
		}()
		count++
	}
	if err := closer.Close(); err != nil {
		log.Fatal(err)
	}

	f, err := os.Create("m3_" + strconv.Itoa(*n) + "k.mprof")
	if err != nil {
		log.Fatal(err)
	}
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}

	log.Println("number of counters:", len(scope.(tally.TestScope).Snapshot().Counters()))
}
