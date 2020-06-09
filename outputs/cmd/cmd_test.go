package cmd

import (
	"github.com/gabrielperezs/goreactor/lib"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewArguments(t *testing.T) {
	var r *lib.Reactor = nil

	c := make(map[string]interface{})
	c["cmd"] = "php"
	c["args"] = []interface{}{"file.php", "first_arg", "second_arg"}

	cmd, err := NewOrGet(r, c)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "", cmd.user)
	assert.Equal(t, "php", cmd.cmd)
	assert.Equal(t, 3, len(cmd.args))
	assert.Equal(t, "file.php", cmd.args[0])
	assert.Equal(t, "first_arg", cmd.args[1])
	assert.Equal(t, "second_arg", cmd.args[2])
}

type Msg struct {
	B []byte
}

func (m *Msg) Body() []byte {
	return m.B
}

func TestJqReplaceWithoutReplacing(t *testing.T) {
	var r *lib.Reactor = nil

	c := make(map[string]interface{})
	c["cmd"] = "php"
	c["args"] = []interface{}{"file.php", "first_arg", "second_arg"}

	cmd, err := NewOrGet(r, c)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "", cmd.user)
	assert.Equal(t, "php", cmd.cmd)
	assert.Equal(t, 3, len(cmd.args))
	assert.Equal(t, "file.php", cmd.args[0])
	assert.Equal(t, "first_arg", cmd.args[1])
	assert.Equal(t, "second_arg", cmd.args[2])

	var msg lib.Msg = &Msg{
		B:   []byte("{\"lang\":\"python3\",\"script\":\"/home/ec2-user/scripts/webbeds-datalake-etl-jobs/webbeds_searches/aggregations/jactravel/jactravel_searches_by_agent.py\"}"),
	}

	var args []string
	for _, parse := range cmd.args {
		args = append(args, cmd.findReplace(msg.Body(), parse))
	}

	assert.Equal(t, 3, len(args))
	assert.Equal(t, "file.php", args[0])
	assert.Equal(t, "first_arg", args[1])
	assert.Equal(t, "second_arg", args[2])
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
		B:   []byte("{\"lang\":\"python3\",\"script\":\"script01\"}"),
	}

	var args []string
	for _, parse := range cmd.args {
		args = append(args, cmd.findReplace(msg.Body(), parse))
	}

	assert.Equal(t, 2, len(args))
	assert.Equal(t, "python3", args[0])
	assert.Equal(t, "script01", args[1])
}