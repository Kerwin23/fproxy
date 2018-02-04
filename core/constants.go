package core

import (
	"strings"
	"time"
)

//代理类型
const (
	Transparent   = iota //透明
	Anonymous            //匿名
	HighAnonymous        //高匿
)

const (
	PROXY_SOURCE_CRAW = "craw"
	PROXY_SOURCE_SCAN = "scan"
)

const (
	PROXY_CHECK_QUEUE   = "proxy:q:check"
	PROXY_POOL_VALID    = "proxy:pool:valid"
	PROXY_POOL_HISTORY  = "proxy:pool:history"
	PROXY_COUNT_SCAN    = "proxy:count:scan:"
	PROXY_COUNT_CRAW    = "proxy:count:craw"
	PROXY_COUNT_HISTORY = "proxy:count:history"
)

func GetProxyTimeKey(src string) string {
	now := time.Now()
	timestr := now.Format("20060102")
	if strings.HasSuffix(src, ":") {
		return src + timestr
	}
	return src + ":" + timestr
}
