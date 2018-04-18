package lib

type Input interface {
	Put(m *Msg) error
	Delete(m *Msg) error
}

type Output interface {
	Run(a *Msg) error
}
