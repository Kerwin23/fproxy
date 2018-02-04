package builder

import (
	core "fproxy/core"
)

//处理结果
const (
	OK = iota
	NotConnected
	NotProxy
)

type Builder interface {
	build() (int, core.Proxy)
}
