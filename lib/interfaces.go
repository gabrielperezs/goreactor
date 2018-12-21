package lib

// Input is the interface for the Input plugins
type Input interface {
	Put(m Msg) error
	Delete(m Msg) error
	Exit()
}

// Output is the interface for the Output plugins
type Output interface {
	MatchConditions(a Msg) error
	Run(rl ReactorLog, a Msg) error
	Exit()
}
