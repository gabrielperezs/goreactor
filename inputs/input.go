package inputs

import (
	"fmt"
	"strings"

	"github.com/gabrielperezs/goreactor/inputs/sqs"
	"github.com/gabrielperezs/goreactor/lib"
	"github.com/gabrielperezs/goreactor/reactor"
)

// Get will start the input plugins
func Get(r *reactor.Reactor, cfg any) (lib.Input, error) {

	var c map[string]any
	var ok bool

	if c, ok = cfg.(map[string]any); !ok {
		return nil, fmt.Errorf("Can't read the configuration (hint: Input)")
	}

	for k, v := range c {
		switch strings.ToLower(k) {
		case "input":
			switch strings.ToLower(v.(string)) {
			case "sqs":
				return sqs.NewOrGet(r, c)
			default:
				return nil, fmt.Errorf("Plugin don't exists: %s", k)
			}
		}
	}

	return nil, fmt.Errorf("Unknown error")
}
