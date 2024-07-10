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
) (string, error) {
	l := lua.NewState()
	defer l.Close()
	luajson.Preload(l)
	l.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)

	l.SetGlobal("ResponseBody", lua.LString(rawBody))
	l.SetGlobal("ResponseStatusCode", lua.LNumber(statusCode))

	if err := l.DoString(script); err != nil {
		return "", err
	}

	vv := l.Get(-1)

	return vv.String(), nil
}
