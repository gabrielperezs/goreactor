package logstreams

import (
	"fmt"
	"strings"

	"github.com/gabrielperezs/goreactor/lib"
	"github.com/gabrielperezs/goreactor/logstreams/awsfirehose"
	"github.com/gabrielperezs/goreactor/logstreams/localstream"
)

// Get will start the output plugins
func Get(cfg any) (lib.LogStream, error) {

	var c map[string]any
	var ok bool

	if c, ok = cfg.(map[string]any); !ok {
		return nil, fmt.Errorf("Can't read the configuration (hint: Logstreams)")
	}

	for k, v := range c {
		switch strings.ToLower(k) {
		case "logstream":
			switch strings.ToLower(v.(string)) {
			case "firehose":
				return awsfirehose.NewOrGet(c)
			case "stdout":
				return localstream.LogStream{}, nil
			case "", "none":
				return nil, fmt.Errorf("WARNING: logstream is disabled")
			default:
				return nil, fmt.Errorf("ERROR: logstream plugin %s doesn't exist", k)
			}
		default:
			return nil, fmt.Errorf("Plugin don't exists: %s", k)
		}
	}

	return nil, fmt.Errorf("Unknown error")
}
