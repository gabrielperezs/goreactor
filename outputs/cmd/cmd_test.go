package cmd

import (
	"github.com/gabrielperezs/goreactor/lib"
	"github.com/stretchr/testify/assert"
	"testing"
)

type Msg struct {
	B []byte
}

func (m *Msg) Body() []byte {
	return m.B
}

func TestJqReplaceActuallyReplacing(t *testing.T) {
	var r *lib.Reactor = nil

	c := make(map[string]interface{})
	c["cmd"] = "cmd_name"
	c["args"] = []interface{}{"$.lang", "$.script"}

	cmd, err := NewOrGet(r, c)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "", cmd.user)
	assert.Equal(t, "cmd_name", cmd.cmd)
	assert.Equal(t, 2, len(cmd.args))
	assert.Equal(t, "$.lang", cmd.args[0])
	assert.Equal(t, "$.script", cmd.args[1])

	var msg lib.Msg = &Msg{
		B: []byte("{\"lang\":\"python3\",\"script\":\"script01\"}"),
	}

	var args = cmd.getReplacedArguments(msg.Body())

	assert.Equal(t, 2, len(args))
	assert.Equal(t, "python3", args[0])
	assert.Equal(t, "script01", args[1])
}

func TestFindReplaceReturningSlice(t *testing.T) {
	var r *lib.Reactor = nil

	c := make(map[string]interface{})
	c["cmd"] = "cmd_name"
	c["args"] = []interface{}{"$.lang", "$.script", "$.args..."}

	cmd, err := NewOrGet(r, c)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "", cmd.user)
	assert.Equal(t, "cmd_name", cmd.cmd)
	assert.Equal(t, 3, len(cmd.args))
	assert.Equal(t, "$.lang", cmd.args[0])
	assert.Equal(t, "$.script", cmd.args[1])
	assert.Equal(t, "$.args...", cmd.args[2])

	var msg lib.Msg = &Msg{
		B: []byte(`{"lang":"python3","script":"script01","args":["third", "fourth"]}`),
	}

	var args = cmd.getReplacedArguments(msg.Body())

	assert.Equal(t, 4, len(args))
	assert.Equal(t, "python3", args[0])
	assert.Equal(t, "script01", args[1])
	assert.Equal(t, "third", args[2])
	assert.Equal(t, "fourth", args[3])
}
