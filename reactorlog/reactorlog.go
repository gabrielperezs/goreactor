package reactorlog

type ReactorLog interface {
	Write(b []byte) (int, error)
	Start(pid int, s string)
	SetLabel(string)
	Done(error)
}
