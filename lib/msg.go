package lib

// Msg internal message interface that is share betwean Input plugins and Output plugins
type Msg interface {
	Body() []byte
	CreationTimestampMilliseconds() int64
	GetHash() string
}
