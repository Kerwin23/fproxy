package check

import (
	"encoding/json"
	"fmt"
	core "fproxy/core"
	"fproxy/httputil"
	store "fproxy/store"
	"github.com/golang/glog"
	"net/http"
	"strconv"
	"time"
)

type CheckResult struct {
	Valid bool
}

type HistoryWorker struct {
	UserAgent  string
	Redis      *store.RedisManager
	ProxyChan  chan core.Proxy
	ResultChan chan CheckResult
	CheckUrls  []string
}

func (h *HistoryWorker) DoWork() {
	for {
		proxy := <-h.ProxyChan
		checkValue := false
		for i, checkUrl := range h.CheckUrls {
			proxyAddr := proxy.Ip + ":" + fmt.Sprintf("%d", proxy.Port)
			checkValue := httputil.HeadForCheck(checkUrl, proxyAddr, nil, http.StatusOK)
			if checkValue || i > 3 {
				break
			}
		}
		checkResult := CheckResult{Valid: checkValue}
		h.ResultChan <- checkResult

	}
}

type HistoryChecker struct {
	Redis      *store.RedisManager
	ProxyChan  chan core.Proxy
	ResultChan chan CheckResult
	Workers    []*HistoryWorker
}

func (h *HistoryChecker) CheckAll() {
	for {
		proxys, err := h.Redis.Smembers(core.PROXY_POOL_HISTORY)
		if err != nil {
			glog.Errorln("get history proxy from redis error: ", err)
			return
		}
		if proxys == nil {
			return
		}
		for _, bsproxy := range proxys {
			proxy := core.Proxy{}
			err = json.Unmarshal(bsproxy, &proxy)
			if err != nil {
				glog.Errorln("history check proxy unmarshal proxy error: ", err)
				continue
			}
			h.ProxyChan <- proxy
		}
		proxylen := len(proxys)
		success := 0
		for i := 0; i < proxylen; i++ {
			result := <-h.ResultChan
			if result.Valid {
				success++
			}
		}
		historyCount := strconv.Itoa(success)
		h.Redis.Set(core.PROXY_COUNT_HISTORY, historyCount)
		time.Sleep(30 * time.Second)
	}
}

func NewHistoryChecker(redis *store.RedisManager, nWorkers, checkSize int, userAgent string, checkUrls []string) *HistoryChecker {
	if checkSize <= 0 {
		checkSize = 100
	}
	proxyChan := make(chan core.Proxy, checkSize)
	resultChan := make(chan CheckResult, checkSize)
	if nWorkers <= 0 {
		nWorkers = 10
	}
	workers := make([]*HistoryWorker, nWorkers)
	for i := 0; i < nWorkers; i++ {
		worker := &HistoryWorker{Redis: redis, UserAgent: userAgent, CheckUrls: checkUrls, ProxyChan: proxyChan, ResultChan: resultChan}
		go worker.DoWork()
		workers[i] = worker
	}
	return &HistoryChecker{Redis: redis, ProxyChan: proxyChan, ResultChan: resultChan, Workers: workers}
}
