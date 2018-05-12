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
	version = "0.2"
)

type Config struct {
	LogFile string
	Reactor []interface{}
	r       []*lib.Reactor
}

var (
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

	<-chMain
}

func exit() {
	for _, r := range conf.r {
		r.Exit()
	}
	chMain <- true
}

func reload() {

	for _, r := range conf.r {
		r.Exit()
	}
	conf.r = nil

	var c *Config
	if _, err := toml.DecodeFile(configFile, &c); err != nil {
		log.Printf("ERROR reading config file %s: %s", configFile, err)
		return
	}
	mu.Lock()
	conf = *c
	mu.Unlock()

	lib.LogReload(conf.LogFile)

	//var err error
	for _, r := range conf.Reactor {
		var err error

		nr := lib.NewReactor(r)
		nr.I, err = inputs.Get(nr, r)
		if err != nil {
			log.Printf("ERROR: %s %s", r, err)
			continue
		}

		nr.O, err = outputs.Get(nr, r)
		if err != nil {
			log.Printf("ERROR: %s %s", r, err)
			continue
		}

		nr.Start()
		conf.r = append(conf.r, nr)
	}
}

func sing() {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		switch <-sigs {
		case syscall.SIGHUP:
			log.Printf("Reloading..")
			reload()
		default:
			log.Printf("Closing...")
			exit()
			chMain <- true
		}
	}
}
