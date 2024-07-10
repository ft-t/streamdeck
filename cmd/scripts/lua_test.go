package scripts_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ft-t/streamdeck/cmd/scripts"
)

var jsonString = `{"alerts":[{"labels":{"alertname":"Watchdog"}},{"labels":{"alertname":"CriticalAlert"}},{"labels":{"alertname":"AnotherAlert"}}]}`

func TestLua(t *testing.T) {
	executor := scripts.NewLua()

	resp, err := executor.Execute(context.TODO(), `
   	    message = "Hello, world!" .. tostring(_G.ResponseStatusCode)
		return message .. _G.ResponseBody`, jsonString, 200, nil, nil)
	assert.NoError(t, err)

	assert.Equal(t, "Hello, world!200{\"alerts\":[{\"labels\":{\"alertname\":\"Watchdog\"}},{\"labels\":{\"alertname\":\"CriticalAlert\"}},{\"labels\":{\"alertname\":\"AnotherAlert\"}}]}", resp)
}

func TestLuaWithHeadersAndTemplate(t *testing.T) {
	executor := scripts.NewLua()

	resp, err := executor.Execute(context.TODO(), `
   	    message = "Hello, world!" .. tostring(_G.Headers["SomeHeader"]) .. tostring(_G.TemplateVariables["key1"])
		return message`,
		jsonString,
		200,
		map[string]string{"Content-Type": "application/json", "SomeHeader": "SomeValue"},
		map[string]string{"key1": "value1", "key2": "value2"},
	)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, world!SomeValuevalue1", resp)
}
