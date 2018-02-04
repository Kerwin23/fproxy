package server

import (
	"fproxy/builder"
	ictx "github.com/kataras/iris/context"
)

type VPSHandler struct {
	VPS *builder.VPS
}

func (v *VPSHandler) HandleAaddVPS(ctx ictx.Context) {
	params := ctx.Params()
	name := params.Get("vps")
	ip := params.Get("ip")
	port, _ := params.GetInt("port")
	v.VPS.AddVPS(name, ip, port)
	ctx.WriteString("ok")
}
