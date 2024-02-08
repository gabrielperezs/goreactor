package noopreactorlog

type NoopReactorLog struct{}

func (NoopReactorLog) Start(pid int, s string) {}
func (NoopReactorLog) SetLabel(string)         {}
func (NoopReactorLog) SetHash(string)          {}
func (NoopReactorLog) Done(error)              {}
func (NoopReactorLog) Write(b []byte) (int, error) {
	return len(b), nil
}
