package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v3"
)

// DaemonConfig is only used for options specific to daemon mode.
// Use CLI arguments for options that are common between CLI/Daemon.
type DaemonConfig struct {
	RunAddress    string   `yaml:"run_address"`
	CheckInterval uint32   `yaml:"check_interval"`
	TCPAddresses  []string `yaml:"tcp_addresses"`
}

func parseConfigFile(path string) *DaemonConfig {
	var settings DaemonConfig
	yamlFile, err := os.Open(path)
	if err != nil {
		log.Fatalln("Could not open config file", err)
	}
	defer yamlFile.Close()
	byteValue, _ := io.ReadAll(yamlFile)
	err = yaml.Unmarshal(byteValue, &settings)
	if err != nil {
		log.Fatalln("Error parsing config file", err)
	}

	// Validate addresses.
	for _, addr := range settings.TCPAddresses {
		if _, err := net.ResolveTCPAddr("tcp", addr); err != nil {
			log.Fatalf("Can not resolve '%s': %s\n", addr, err)
		}
	}

	log.Printf("Parsed %s\n", path)
	return &settings
}

type Daemon struct {
	prometheus struct {
		registry            *prometheus.Registry
		checkDurationMetric *prometheus.GaugeVec
		errorCountMetric    *prometheus.CounterVec
	}
	config    *DaemonConfig
	cliArgs   *CLIConfig
	ctxCancel context.CancelFunc
	Ctx       context.Context
}

// Start the HTTP server.
func (d *Daemon) StartMetricsServer() {
	http.Handle("/metrics", promhttp.HandlerFor(
		d.prometheus.registry,
		promhttp.HandlerOpts{Registry: d.prometheus.registry},
	))
	log.Printf("Starting metrics daemon at %s , metrics will be available at /metrics\n", d.config.RunAddress)
	err := http.ListenAndServe(d.config.RunAddress, nil)
	if err != nil {
		log.Fatalln("Failed to start HTTP server", err)
	}
}

// Stop running checker and cancel the daemon context.
func (d *Daemon) Stop() {
	d.ctxCancel()
}

// Run the checker once for all configured TCP addresses.
func (d *Daemon) RunChecker() {
	for _, addr := range d.config.TCPAddresses {
		checker := NewConcurrentChecker(d.cliArgs, addr)
		checkerCtx, cancel := context.WithCancel(context.Background())

		startedAt := time.Now()
		if err := checker.Launch(checkerCtx); err != nil {
			log.Printf("Failed to launch checker for address %s, %v\n", addr, err)
			cancel()
			checker.Stop()
			continue
		}

		// Normally, cancel context after finishing (because we need to close concurrent checker's CheckingLoop),
		// but don't wait before cancelling if the daemon needs to stop.
		select {
		case <-d.Ctx.Done():
			log.Println("Stopping running check")
			cancel()
			checker.Stop()
			return
		case <-checker.Wait():
			cancel()
		}
		duration := time.Since(startedAt)

		if d.cliArgs.Verbose {
			log.Printf("Finished %d/%d checks in %s for addr %s\n",
				checker.Count(CRequest), d.cliArgs.Requests, duration, addr)
			log.Printf("Succeed: %d\n", checker.Count(CSucceed))
			log.Printf("Errors: connect %d, timeout %d, other %d\n",
				checker.Count(CErrConnect), checker.Count(CErrTimeout), checker.Count(CErrOther))
		}

		d.prometheus.checkDurationMetric.
			WithLabelValues(addr, fmt.Sprintf("%d", d.cliArgs.Requests)).Set(float64(duration.Milliseconds()))

		d.prometheus.errorCountMetric.
			WithLabelValues("connect", addr, fmt.Sprintf("%d", d.cliArgs.Requests)).Add(float64(checker.Count(CErrConnect)))
		d.prometheus.errorCountMetric.
			WithLabelValues("timeout", addr, fmt.Sprintf("%d", d.cliArgs.Requests)).Add(float64(checker.Count(CErrTimeout)))
		d.prometheus.errorCountMetric.
			WithLabelValues("other", addr, fmt.Sprintf("%d", d.cliArgs.Requests)).Add(float64(checker.Count(CErrOther)))

		checker.Stop()
	}

}

func NewDaemon(cliArgs *CLIConfig) Daemon {
	ctx, cancel := context.WithCancel(context.Background())
	d := Daemon{
		config:    parseConfigFile(cliArgs.ConfigFile),
		cliArgs:   cliArgs,
		Ctx:       ctx,
		ctxCancel: cancel,
	}

	d.prometheus.registry = prometheus.NewRegistry()
	d.prometheus.checkDurationMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tcpcheck_duration",
			Help: "TCP Check duration in ms, partitioned by destination address and number of requests per check.",
		},
		[]string{
			// Which TCP address was tested
			"destination",
			// Requests per check
			"requests_per_check",
		},
	)
	d.prometheus.errorCountMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "error_count",
			Help: "Number of errors occurred, partitioned by error type, destination address and number of requests per check.",
		},
		[]string{
			// What type of error
			"error_type",
			// Which TCP address was tested
			"destination",
			// Requests per check
			"requests_per_check",
		},
	)
	d.prometheus.registry.MustRegister(d.prometheus.checkDurationMetric)
	d.prometheus.registry.MustRegister(d.prometheus.errorCountMetric)
	return d
}

// Run the program in Daemon Mode.
func daemonMode(cliArgs *CLIConfig) {
	d := NewDaemon(cliArgs)

	go gracefulDaemonShutdown(&d)

	// Note that the next run won't start unless the previous iteration has finished or timed-out.
	ticker := time.NewTicker(time.Duration(d.config.CheckInterval) * time.Second)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		for {
			select {
			case <-d.Ctx.Done():
				log.Println("Quitting")
				os.Exit(0)
			case <-ticker.C:
				if cliArgs.Verbose {
					log.Println("New tick, running tcp-check on all addresses again")
				}
				d.RunChecker()
			}
		}
	}()

	d.StartMetricsServer()

	wg.Wait()
}

func gracefulDaemonShutdown(d *Daemon) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	log.Println("Got quit signal")
	d.Stop()
}
