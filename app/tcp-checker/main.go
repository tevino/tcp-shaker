package main

import (
	"flag"
	"log"
	"net"
	"time"
)

// CLIConfig contains all available CLI options.
type CLIConfig struct {
	// General
	Timeout     time.Duration
	Requests    int
	Concurrency int
	Verbose     bool

	// Daemon mode
	Daemon     bool
	ConfigFile string

	// CLI / one-off mode
	Addr string
}

func parseCLIConfig() *CLIConfig {
	var conf CLIConfig
	var timeoutMS int

	// Flag definitions
	// - General
	flag.IntVar(&timeoutMS, "t", 1000, "Timeout in millisecond for the whole checking process(domain resolving is included)")
	flag.IntVar(&conf.Requests, "n", 1, "Number of requests to perform")
	flag.IntVar(&conf.Concurrency, "c", 1, "Number of checks to perform simultaneously")
	flag.BoolVar(&conf.Verbose, "v", false, "Print more logs e.g. error detail")
	// - CLI / one-off mode
	flag.StringVar(&conf.Addr, "a", "google.com:80", "TCP address to test")
	// - Daemon mode
	flag.BoolVar(&conf.Daemon, "d", false, "Run in daemon mode and expose a prometheus compatible metrics")
	flag.StringVar(&conf.ConfigFile, "f", "", "Path to config file for daemon mode")

	// Parse flags
	flag.Parse()
	if _, err := net.ResolveTCPAddr("tcp", conf.Addr); err != nil {
		log.Fatalf("Can not resolve '%s': %s\n", conf.Addr, err)
	}
	conf.Timeout = time.Duration(timeoutMS) * time.Millisecond

	if conf.Daemon && conf.ConfigFile == "" {
		log.Fatalln("Config should be provided for running daemon mode")
	}

	return &conf
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()

	cliArgs := parseCLIConfig()

	if cliArgs.Daemon {
		daemonMode(cliArgs)
	} else {
		log.SetFlags(0)
		cliMode(cliArgs)
	}
}
