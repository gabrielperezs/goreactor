package main

import (
	"flag"
	"log"
	"os"
	"sync"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/gabrielperezs/goreactor/inputs"
	"github.com/gabrielperezs/goreactor/lib"
	"github.com/gabrielperezs/goreactor/outputs"
)

const (
	version = "0.1"
)

type Config struct {
	Reactor []interface{}
	r       []*lib.Reactor
}

var (
	conf       Config
	configFile string
	mu         sync.Mutex
	chSign     = make(chan os.Signal, 10)
	chMain     = make(chan bool)
)

func main() {
	flag.StringVar(&configFile, "config", "config.conf", "Configuration file")
	flag.Parse()

	log.Printf("Starting v%s", version)

	go sing()
	reload()
}

func reload() {

	for _, r := range conf.r {
		log.Printf("Close", r)
	}

	var c *Config
	if _, err := toml.DecodeFile(configFile, &c); err != nil {
		log.Printf("ERROR reading config file %s: %s", configFile, err)
		return
	}
	mu.Lock()
	conf = *c
	mu.Unlock()

	//var err error
	for _, r := range conf.Reactor {
		var err error

		nr := lib.NewReactor()
		conf.r = append(conf.r, nr)

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
	}

	<-chMain
	log.Printf("Close")
}

func sing() {
	for {
		switch <-chSign {
		case syscall.SIGHUP:
			log.Printf("Reloading..")
			reload()
		default:
			log.Printf("Closing...")
			chMain <- true
		}
	}
}
