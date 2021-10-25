package localstream

import (
	"log"
)

type LogStream struct {}


func (LogStream) Send(b []byte) {
	log.Print(string(b))
}

func (LogStream) Exit() {}
