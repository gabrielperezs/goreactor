package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/gabrielperezs/goreactor/inputs"
	"github.com/gabrielperezs/goreactor/lib"
	"github.com/gabrielperezs/goreactor/outputs"
)

const (
	version = "1.1"
)

// Config contains all the configuration parameters needed
// by all the plugins and filled with the TOML parser
type Config struct {
	LogFile string
	Reactor []interface{}
}

var (
	running    []*lib.Reactor
	conf       Config
	configFile string
	debug      bool
	mu         sync.Mutex
	chMain     = make(chan bool)
)

func main() {
	flag.StringVar(&configFile, "config", "config.conf", "Configuration file")
	flag.BoolVar(&debug, "debug", false, "Debug mode")
	flag.Parse()

	if !debug {
		log.SetFlags(0)
	}
	log.Printf("Starting v%s", version)

	go sing()
	reload()
	start()

	<-chMain
}

func start() {
	for _, r := range conf.Reactor {
		nr := lib.NewReactor(r)

		var err error
		nr.I, err = inputs.Get(nr, r)
		if err != nil {
			log.Printf("ERROR: %s %s", r, err)
		}

		nr.O, err = outputs.Get(nr, r)
		if err != nil {
			log.Printf("ERROR: %s %s", r, err)
		}

		nr.Start()
		running = append(running, nr)
	}
}

func exit() {
	for _, r := range running {
		r.Exit()
	}
	chMain <- true
}

func restart() {
	for _, r := range running {
		r.Exit()
	}
	running = nil
	start()
}

func reload() {
	var c *Config
	if _, err := toml.DecodeFile(configFile, &c); err != nil {
		log.Printf("ERROR reading config file %s: %s", configFile, err)
		return
	}
	mu.Lock()
	conf = *c
	mu.Unlock()

	lib.LogReload(conf.LogFile)
}

func sing() {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGUSR1)

	for {
		switch <-sigs {
		case syscall.SIGHUP:
			log.Printf("Rotate logs")
			reload()
		case syscall.SIGUSR1:
			log.Printf("Full restart")
			reload()
			restart()
		default:
			log.Printf("Exiting...")
			exit()
			chMain <- true
		}
	}
}
