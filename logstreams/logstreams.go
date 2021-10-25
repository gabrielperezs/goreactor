package logstreams

import (
	"fmt"
	"strings"

	"github.com/gabrielperezs/goreactor/lib"
	"github.com/gabrielperezs/goreactor/logstreams/awsfirehose"
	"github.com/gabrielperezs/goreactor/logstreams/localstream"
)

// Get will start the output plugins
func Get(cfg interface{}) (lib.LogStream, error) {

	var c map[string]interface{}
	var ok bool

	if c, ok = cfg.(map[string]interface{}); !ok {
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
			}
		default:
			return nil, fmt.Errorf("Plugin don't exists: %s", k)
		}
	}

	return nil, fmt.Errorf("Unknown error")
}
