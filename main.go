package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/gabrielperezs/goreactor/inputs"
	"github.com/gabrielperezs/goreactor/lib"
	"github.com/gabrielperezs/goreactor/logstreams"
	"github.com/gabrielperezs/goreactor/outputs"
)

const (
	version = "1.3.1"
)

// Config contains all the configuration parameters needed
// by all the plugins and filled with the TOML parser
type Config struct {
	LogStream interface{}
	Reactor   []interface{}
}

var (
	running    []*lib.Reactor
	conf       Config
	configFile string
	configDir  string
	debug      bool
	mu         sync.Mutex
	chMain     = make(chan bool)
)

func main() {
	flag.StringVar(&configFile, "config", "config.conf", "Configuration file")
	flag.StringVar(&configDir, "d", "", "Configuration directory")
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

	hostname, _ := os.Hostname()

	lg, err := logstreams.Get(conf.LogStream)
	if err != nil {
		log.Printf("ERR: %s", err.Error())
	}

	for _, r := range conf.Reactor {
		nr := lib.NewReactor(r)
		nr.SetLogStreams(lg)
		nr.SetHostname(hostname)

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
	c, err := readConfig(configFile, configDir)
	if err != nil {
		log.Printf("ERROR reading config file or directory %s,%s: %s", configFile, configDir, err)
		return
	}
	mu.Lock()
	conf = *c
	mu.Unlock()
}

func sing() {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		switch <-sigs {
		case syscall.SIGHUP:
			log.Printf("Rotate logs")
			reload()
			restart()
		case syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT, os.Interrupt:
			log.Printf("Exiting...")
			exit()
			chMain <- true
		default:
		}
	}
}

func readConfig(configFile, configDir string) (c *Config, err error) {
	c = &Config{
		Reactor: make([]interface{}, 0),
	}

	if configDir == "" {
		//Configuration is a file
		_, err = toml.DecodeFile(configFile, &c)
		return c, err
	}

	//Configuration is a directory
	fileInfo, err := os.Stat(configDir)
	if err != nil {
		return nil, err
	}

	if !fileInfo.IsDir() {
		return c, fmt.Errorf("Configuration directory is not a directory: %s", configDir)
	}

	files, err := ioutil.ReadDir(configDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if filepath.Ext(file.Name()) != ".conf" {
			continue
		}

		pathTemp := fmt.Sprintf("%s/%s", configDir, file.Name())

		var configTemp *Config
		if _, err := toml.DecodeFile(pathTemp, &configTemp); err != nil {
			log.Printf("ERROR reading config file %s: %s", pathTemp, err)
			continue
		}

		for _, reactor := range configTemp.Reactor {
			c.Reactor = append(c.Reactor, reactor)
		}

		// Read first LogStream only
		if c.LogStream == nil && configTemp.LogStream != nil {
			c.LogStream = configTemp.LogStream
		}
	}
	return c, nil
}
