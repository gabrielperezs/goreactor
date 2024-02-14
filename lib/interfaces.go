package lib

import (
	"context"
	"time"

	"github.com/gabrielperezs/goreactor/reactorlog"
)

// Input is the interface for the Input plugins
type Input interface {
	KeepAlive(context.Context, time.Duration, Msg) error
	Done(Msg, bool) // Input was processed and don't need to keep it as pending
	Stop()          // Stop accepting input
	Exit()          // Exit from the loop
}

// Output is the interface for the Output plugins
type Output interface {
	MatchConditions(a Msg) error
	Run(ctx context.Context, rl reactorlog.ReactorLog, a Msg) error
	Exit()
}

// LogStreams is the inteface to send logs to stram services
type LogStream interface {
	Send(b []byte)
	Exit()
}
