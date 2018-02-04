package server

import (
	"github.com/kataras/iris"
	ictx "github.com/kataras/iris/context"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
	"strconv"
)

type FProxyServer struct {
	app *iris.Application
}

func (f *FProxyServer) Init() {
	app := iris.New()
	f.app = app
	f.app.Use(recover.New())
	f.app.Use(logger.New())
}

func (f *FProxyServer) DoPost(path string, handler func(ictx.Context)) {
	f.app.Post(path, handler)
}

func (f *FProxyServer) DoGet(path string, handler func(ictx.Context)) {
	f.app.Get(path, handler)
}

func (f *FProxyServer) Run(host string, port int) {
	addr := host + ":" + strconv.Itoa(port)
	f.app.Run(iris.Addr(addr), iris.WithoutServerError(iris.ErrServerClosed))
}

func NewFProxyServer() *FProxyServer {
	svr := &FProxyServer{}
	return svr
}
