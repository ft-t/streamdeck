package scripts

import (
	"context"
	"net/http"

	"github.com/cjoudrey/gluahttp"
	lua "github.com/yuin/gopher-lua"
	luajson "layeh.com/gopher-json"
)

type Lua struct {
}

func NewLua() *Lua {
	return &Lua{}
}

func (e *Lua) Execute(
	_ context.Context,
	script string,
	rawBody string,
	statusCode int,
	headers map[string]string,
	templateVariables map[string]string,
) (string, error) {
	l := lua.NewState()
	defer l.Close()
	luajson.Preload(l)
	l.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)

	l.SetGlobal("ResponseBody", lua.LString(rawBody))
	l.SetGlobal("ResponseStatusCode", lua.LNumber(statusCode))

	luaHeaders := l.NewTable()
	for k, v := range headers {
		l.SetTable(luaHeaders, lua.LString(k), lua.LString(v))
	}
	l.SetGlobal("Headers", luaHeaders)

	luaTemplateVariables := l.NewTable()
	for k, v := range templateVariables {
		l.SetTable(luaTemplateVariables, lua.LString(k), lua.LString(v))
	}
	l.SetGlobal("TemplateVariables", luaTemplateVariables)

	if err := l.DoString(script); err != nil {
		return "", err
	}

	vv := l.Get(-1)

	return vv.String(), nil
}
