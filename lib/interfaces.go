package lib

type Input interface {
	Put(m *Msg) error
	Delete(m *Msg) error
	Exit()
}

type Output interface {
	Run(rl ReactorLog, a *Msg) error
	Exit()
}
