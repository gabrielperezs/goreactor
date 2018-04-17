package inputs

import (
	"fmt"
	"strings"

	"github.com/gabrielperezs/goreactor/inputs/sqs"
	"github.com/gabrielperezs/goreactor/lib"
)

func Get(r *lib.Reactor, cfg interface{}) (lib.Input, error) {

	var c map[string]interface{}
	var ok bool

	if c, ok = cfg.(map[string]interface{}); !ok {
		return nil, fmt.Errorf("Can't read the configuration")
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

	return nil, fmt.Errorf("Unkown error")
}
