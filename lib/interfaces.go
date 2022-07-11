package lib

import "github.com/gabrielperezs/goreactor/reactorlog"

// Input is the interface for the Input plugins
type Input interface {
	Put(m Msg) error
	Delete(m Msg) error
	Stop() // Stop accepting input
	Exit() // Exit from the loop
}

// Output is the interface for the Output plugins
type Output interface {
	MatchConditions(a Msg) error
	Run(rl reactorlog.ReactorLog, a Msg) error
	Exit()
}

// LogStreams is the inteface to send logs to stram services
type LogStream interface {
	Send(b []byte)
	Exit()
}
