package cmd

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/gabrielperezs/goreactor/lib"
	"github.com/savaki/jq"
)

var (
	maximumCmdTimeLive = 10 * time.Minute
)

// Cmd is the command struct that will be executed after recive the order
// from the input plugins
type Cmd struct {
	r        *lib.Reactor
	cmd      string
	args     []string
	argsjson bool
	cond     map[string]string
}

// NewOrGet create the command struct and fill the parameters needed from the
// config data.
func NewOrGet(r *lib.Reactor, c map[string]interface{}) (*Cmd, error) {

	o := &Cmd{
		r:    r,
		cond: make(map[string]string),
	}

	for k, v := range c {
		switch strings.ToLower(k) {
		case "cmd":
			o.cmd = v.(string)
		case "args":
			for _, n := range v.([]interface{}) {
				o.args = append(o.args, n.(string))
			}
		case "argsjson":
			o.argsjson = v.(bool)
		case "cond":
			for _, v := range v.([]interface{}) {
				for nk, nv := range v.(map[string]interface{}) {
					o.cond[nk] = nv.(string)
				}

			}
		}
	}

	return o, nil
}

// MatchConditions is a filter to replace the variables (usually commands arguments)
// that are comming from the Input message
func (o *Cmd) MatchConditions(msg lib.Msg) error {
	if !o.argsjson {
		return nil
	}

	for k, v := range o.cond {
		if strings.HasPrefix(k, "$.") {
			op, _ := jq.Parse(k[1:]) // create an Op
			value, _ := op.Apply(msg.Body())
			nv := strings.Trim(string(value), "\"")

			if nv != v {
				return lib.ErrInvalidMsgForPlugin
			}
		}
	}
	return nil
}

// Run will execute the binary command that was defined in the config.
// In this function we also define the OUT and ERR data destination of
// the command.
func (o *Cmd) Run(rl lib.ReactorLog, msg lib.Msg) error {

	var args []string

	if o.argsjson {
		for _, parse := range o.args {
			if strings.Contains(parse, "$.") {
				newParse := parse
				for _, argValue := range strings.Split(parse, "$.") {
					if argValue == "" {
						continue
					}
					op, _ := jq.Parse("." + argValue) // create an Op
					value, _ := op.Apply(msg.Body())
					newParse = strings.Replace(newParse, "$."+argValue, strings.Trim(string(value), "\""), -1)
				}
				args = append(args, newParse)
			} else {
				args = append(args, parse)
			}
		}
	} else {
		args = o.args
	}

	ctx, cancel := context.WithTimeout(context.Background(), maximumCmdTimeLive)
	defer cancel()

	var c *exec.Cmd
	if len(args) > 0 {
		c = exec.CommandContext(ctx, o.cmd, args...)
	} else {
		c = exec.CommandContext(ctx, o.cmd)
	}

	c.Stdout = rl
	c.Stderr = rl

	runlog := fmt.Sprintf("RUN: %s %s", o.cmd, strings.Join(args, " "))
	log.Println(runlog)
	rl.WriteStrings(runlog)
	if err := c.Run(); err != nil {
		// This will fail after timeout.
		rl.WriteStrings(fmt.Sprintf("ERROR: %s", err))
		return err
	}

	return nil
}

// Exit will finish the command // TODO
func (o *Cmd) Exit() {

}
