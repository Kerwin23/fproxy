package processor

import (
	"encoding/json"
	"fmt"
	"fproxy/core"
	"fproxy/httputil"
	"fproxy/store"
	"github.com/golang/glog"
	"math/rand"
)

const KEY_PROXY_CACHE = "proxy:valid:"

type HttpProcessor struct {
	UserAgent     string
	CheckRequests []CheckRequest
	RedisManager  *store.RedisManager
	RequestRand   *rand.Rand
}

func (h *HttpProcessor) SetCheckRequests(checkRequests []CheckRequest) {
	h.CheckRequests = checkRequests
}

func (h *HttpProcessor) Process(proxy core.Proxy) int {
	request := h.selectRequest()
	result := h.processOne(request, proxy.Ip, proxy.Port)
	if result == SUCCESS {
		return result
	}
	return FAIL
}

func (h *HttpProcessor) selectRequest() CheckRequest {
	reqLen := len(h.CheckRequests)
	index := h.RequestRand.Intn(reqLen)
	return h.CheckRequests[index]
}

func (h *HttpProcessor) processOne(request CheckRequest, ip string, port int) int {
	userAgent := request.UserAgent
	if userAgent == "" {
		userAgent = h.UserAgent
	}
	headers := make(map[string]string)
	headers["User-Agent"] = userAgent
	proxy := ip + ":" + fmt.Sprintf("%d", port)
	isProxy := httputil.GetForCheck(request.Url, proxy, request.Word, headers, request.MaxLength)
	if isProxy {
		return SUCCESS
	}
	return FAIL
}

func (h *HttpProcessor) OnSuccess(proxy core.Proxy) {
	glog.Infoln("scan find proxy: ", proxy)
	bs, err := json.Marshal(proxy)
	if err != nil {
		glog.Errorln("http processor marshal proxy error: ", err)
		return
	}
	h.RedisManager.Rpush(core.PROXY_CHECK_QUEUE, string(bs))
}

func (h *HttpProcessor) OnFail(proxy core.Proxy) {
	glog.Infoln("check proxy fail: invalid proxy[", proxy.Ip, ",", proxy.Port, ",", proxy.Source, "]")
}
