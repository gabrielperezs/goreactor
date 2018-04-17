package cmd

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gabrielperezs/goreactor/lib"
	"github.com/savaki/jq"
)

type Cmd struct {
	r        *lib.Reactor
	cmd      string
	args     []string
	argsjson bool
}

func NewOrGet(r *lib.Reactor, c map[string]interface{}) (*Cmd, error) {

	o := &Cmd{
		r: r,
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
		}
	}

	return o, nil
}

func (o *Cmd) Run(i interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var args []string

	if o.argsjson {
		var b []byte
		switch i.(type) {
		case []byte:
			b = i.([]byte)
		case string:
			b = []byte(i.(string))
		default:
			log.Print("ERROR: invalid format")
			return
		}
		for _, parse := range o.args {
			if strings.HasPrefix(parse, ".") {
				op, _ := jq.Parse(parse) // create an Op
				value, _ := op.Apply(b)
				args = append(args, string(value))
			} else {
				args = append(args, parse)
			}
		}

		log.Printf("O: %s", string(b))
	} else {
		args = o.args
	}

	var c *exec.Cmd
	if len(args) > 0 {
		c = exec.CommandContext(ctx, o.cmd, args...)
	} else {
		c = exec.CommandContext(ctx, o.cmd)
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stdout
	if err := c.Run(); err != nil {
		// This will fail after 100 milliseconds. The 5 second sleep
		// will be interrupted.
	}
}
