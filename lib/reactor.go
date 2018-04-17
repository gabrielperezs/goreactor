package lib

type Reactor struct {
	I  Input
	O  Output
	Ch chan interface{}
}

func NewReactor() *Reactor {
	r := &Reactor{
		Ch: make(chan interface{}),
	}
	go r.listener()
	return r
}

func (r *Reactor) listener() {

	for m := range r.Ch {
		if r.O == nil {
			continue
		}

		r.O.Run(m)
	}

}
