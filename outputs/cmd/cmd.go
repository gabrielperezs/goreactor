package cmd

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gabrielperezs/goreactor/lib"
	"github.com/savaki/jq"
)

var (
	exitInterval       = 500 * time.Millisecond
	waitExit           = 2 * time.Second
	maximumCmdTimeLive = 10 * time.Minute
)

// Cmd is the command struct that will be executed after recive the order
// from the input plugins
type Cmd struct {
	mu      *sync.Mutex
	r       *lib.Reactor
	cmd     string
	args    []string
	cond    map[string]*regexp.Regexp
	running map[int]*exec.Cmd
}

// NewOrGet create the command struct and fill the parameters needed from the
// config data.
func NewOrGet(r *lib.Reactor, c map[string]interface{}) (*Cmd, error) {
	o := &Cmd{
		mu:      &sync.Mutex{},
		r:       r,
		cond:    make(map[string]*regexp.Regexp),
		running: make(map[int]*exec.Cmd),
	}

	for k, v := range c {
		switch strings.ToLower(k) {
		case "cmd":
			o.cmd = v.(string)
		case "args":
			for _, n := range v.([]interface{}) {
				o.args = append(o.args, n.(string))
			}
		case "cond":
			for _, v := range v.([]interface{}) {
				for nk, nv := range v.(map[string]interface{}) {
					o.cond[nk] = regexp.MustCompile(nv.(string))
				}
			}
		}
	}

	return o, nil
}

// MatchConditions is a filter to replace the variables (usually commands arguments)
// that are coming from the Input message
func (o *Cmd) MatchConditions(msg lib.Msg) error {
	for k, v := range o.cond {
		if strings.HasPrefix(k, "$.") {
			op, _ := jq.Parse(k[1:]) // create an Op
			value, _ := op.Apply(msg.Body())
			nv := bytes.Trim(value, "\"")

			if !v.Match(nv) {
				return lib.ErrInvalidMsgForPlugin
			}
		}
	}
	return nil
}

func (o *Cmd) findReplace(b []byte, s string) string {
	if !strings.Contains(s, "$.") {
		return s
	}

	newParse := s
	for _, argValue := range strings.Split(s, "$.") {
		if argValue == "" {
			continue
		}
		op, _ := jq.Parse("." + argValue) // create an Op
		value, _ := op.Apply(b)
		newParse = strings.Replace(newParse, "$."+argValue, strings.Trim(string(value), "\""), -1)
	}
	return newParse
}

// Run will execute the binary command that was defined in the config.
// In this function we also define the OUT and ERR data destination of
// the command.
func (o *Cmd) Run(rl *lib.ReactorLog, msg lib.Msg) error {
	var args []string

	for _, parse := range o.args {
		args = append(args, o.findReplace(msg.Body(), parse))
	}
	rl.Label = o.findReplace(msg.Body(), o.r.Label)

	ctx, cancel := context.WithTimeout(context.Background(), maximumCmdTimeLive)
	defer cancel()

	cmd := exec.CommandContext(ctx, o.cmd, args...)

	cmd.Stdout = rl
	cmd.Stderr = rl

	rl.Start(o.cmd + " " + strings.Join(args, " "))
	if err := cmd.Start(); err != nil {
		return err
	}

	pid := cmd.Process.Pid
	log.Printf("PID: %v", pid)
	o.mu.Lock()
	o.running[pid] = cmd
	o.mu.Unlock()
	defer func() {
		log.Printf("PID-DONE: %v", pid)
		o.mu.Lock()
		delete(o.running, pid)
		o.mu.Unlock()
	}()

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

// Exit will finish the command
func (o *Cmd) Exit() error {
	o.mu.Lock()
	copyRunning := make(map[int]*exec.Cmd, len(o.running))
	for pid, cmd := range o.running {
		copyRunning[pid] = cmd
	}
	o.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), waitExit)
	defer cancel()
	for {
		for pid, cmd := range copyRunning {
			if cmd == nil || cmd.Process == nil || cmd.ProcessState == nil {
				delete(copyRunning, pid)
			}
		}
		if len(copyRunning) == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			for _, cmd := range copyRunning {
				if cmd == nil || cmd.Process == nil || cmd.ProcessState == nil {
					continue
				}
				cmd.Stderr = os.Stderr
				cmd.Stdout = os.Stdout
				cmd.Process.Kill()
			}
			return ctx.Err()
		default:
		}
		time.Sleep(exitInterval)
	}
}
