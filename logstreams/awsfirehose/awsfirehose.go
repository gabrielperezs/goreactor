package awsfirehose

import (
	"log"
	"reflect"
	"strings"

	firehosePool "github.com/gabrielperezs/streamspooler/firehose"
)

type AWSFirehose struct {
	s *firehosePool.Server
}

func NewOrGet(cfg map[string]interface{}) (*AWSFirehose, error) {
	c := firehosePool.Config{}

	v := reflect.ValueOf(&c)
	ps := v.Elem()
	typeOfS := ps.Type()

	cpCfg := make(map[string]interface{})
	for k, v := range cfg {
		cpCfg[strings.ToLower(k)] = v
	}

	for i := 0; i < ps.NumField(); i++ {
		newValue, ok := cpCfg[strings.ToLower(typeOfS.Field(i).Name)]
		if !ok {
			continue
		}

		f := ps.Field(i)
		if !f.IsValid() || !f.CanSet() {
			continue
		}

		switch ps.Field(i).Type().String() {
		case "bool":
			ps.Field(i).SetBool(newValue.(bool))
		case "string":
			ps.Field(i).SetString(newValue.(string))
		case "float32", "float64":
			switch newValue.(type) {
			case float32:
				ps.Field(i).SetFloat(float64(newValue.(float32)))
			case float64:
				ps.Field(i).SetFloat(newValue.(float64))
			}
		case "int":
			switch newValue.(type) {
			case int:
				ps.Field(i).SetInt(int64(newValue.(int)))
			case int64:
				ps.Field(i).SetInt(newValue.(int64))
			}
		}
	}

	o := &AWSFirehose{
		s: firehosePool.New(c),
	}
	return o, nil
}

func (o *AWSFirehose) Send(b []byte) {
	log.Printf("AWSFirehose: %s", string(b))
	o.s.C <- b
}

func (o *AWSFirehose) Exit() {
	o.s.Exit()
}
