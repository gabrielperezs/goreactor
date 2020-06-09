package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"regexp"
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
	r           *lib.Reactor
	cmd         string
	user        string
	environment []string
	args        []string
	cond        map[string]*regexp.Regexp
}

// NewOrGet create the command struct and fill the parameters needed from the
// config data.
func NewOrGet(r *lib.Reactor, c map[string]interface{}) (*Cmd, error) {

	o := &Cmd{
		r:    r,
		cond: make(map[string]*regexp.Regexp),
	}

	for k, v := range c {
		switch strings.ToLower(k) {
		case "cmd":
			o.cmd = v.(string)
		case "args":
			for _, n := range v.([]interface{}) {
				o.args = append(o.args, n.(string))
			}
		case "user":
			o.user = v.(string)
		case "environs", "environment", "env":
			for _, n := range v.([]interface{}) {
				o.environment = append(o.environment, n.(string))
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

func (o *Cmd) findReplaceReturningSlice(b []byte, s string) []string {
	if !strings.HasPrefix(s, "$.") || !strings.HasSuffix(s, "...") {
		return []string{o.findReplace(b, s)} // Fallback to previous function
	}

	cleanArgValue := s[1 : len(s)-3] // Remove initial $ and final ...
	op, err := jq.Parse(cleanArgValue)
	if err != nil {
		return []string{o.findReplace(b, s)} // Fallback to previous function
	}

	substituted, _ := op.Apply(b)
	var values []string

	json.Unmarshal(substituted, &values)
	return values
}

func (o *Cmd) getReplacedArguments(b []byte) []string {
	var args []string
	for _, parse := range o.args {
		args = append(args, o.findReplaceReturningSlice(b, parse)...)
	}
	return args
}

// Run will execute the binary command that was defined in the config.
// In this function we also define the OUT and ERR data destination of
// the command.
func (o *Cmd) Run(rl *lib.ReactorLog, msg lib.Msg) error {

	var args []string
	args = o.getReplacedArguments(msg.Body())

	rl.Label = o.findReplace(msg.Body(), o.r.Label)

	ctx, cancel := context.WithTimeout(context.Background(), maximumCmdTimeLive)
	defer cancel()

	var c *exec.Cmd
	if len(args) > 0 {
		c = exec.CommandContext(ctx, o.cmd, args...)
	} else {
		c = exec.CommandContext(ctx, o.cmd)
	}

	if o.user != "" {
		err := setUserToCmd(o.user, o.environment, c)
		if err != nil {
			return err
		}
	}

	c.Stdout = rl
	c.Stderr = rl

	rl.Start(o.cmd + " " + strings.Join(args, " "))
	if err := c.Run(); err != nil {
		return err
	}

	return nil
}

// Exit will finish the command // TODO
func (o *Cmd) Exit() {

}
