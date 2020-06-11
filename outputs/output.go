package outputs

import (
	"fmt"
	"strings"

	"github.com/gabrielperezs/goreactor/lib"
	"github.com/gabrielperezs/goreactor/outputs/cmd"
)

// Get will start the output plugins
func Get(r *lib.Reactor, cfg interface{}) (lib.Output, error) {

	var c map[string]interface{}
	var ok bool

	if c, ok = cfg.(map[string]interface{}); !ok {
		return nil, fmt.Errorf("Can't read the configuration (hint: Output)")
	}

	for k, v := range c {
		switch strings.ToLower(k) {
		case "output":
			switch strings.ToLower(v.(string)) {
			case "cmd":
				return cmd.NewOrGet(r, c)
			default:
				return nil, fmt.Errorf("Plugin don't exists: %s", k)
			}
		}
	}

	return nil, fmt.Errorf("Unknown error")
}
