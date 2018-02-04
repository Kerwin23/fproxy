package processor

import (
	"fproxy/core"
)

const (
	FAIL = iota
	SUCCESS
)

type Processor interface {
	Process(proxy core.Proxy) int
	OnSuccess(proxy core.Proxy)
	OnFail(proxy core.Proxy)
}
