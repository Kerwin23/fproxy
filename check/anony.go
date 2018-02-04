package check

import (
	"encoding/json"
	"fmt"
	"fproxy/core"
	"fproxy/httputil"
	"fproxy/store"
	"github.com/golang/glog"
	"time"
)

/*
*高匿检测器
 */
type AnonyChecker struct {
	CheckUrl   string
	Redis      *store.RedisManager
	Workers    []AnonyCheckWorker
	CheckQueue chan core.Proxy
}

/*
*高匿检测工作器，并发操作
 */
type AnonyCheckWorker struct {
	CheckUrl    string
	Redis       *store.RedisManager
	CheckQueue  chan core.Proxy
	MaxBodySize int
}

func NewAnonyChecker(checkUrl string, redis *store.RedisManager, nWorkers, checkSize, maxBodySize int) AnonyChecker {
	if checkSize < 1 {
		checkSize = 100
	}
	checkQueue := make(chan core.Proxy, checkSize)
	if nWorkers < 1 {
		nWorkers = 10
	}
	workers := make([]AnonyCheckWorker, nWorkers)
	for i := 0; i < nWorkers; i++ {
		worker := AnonyCheckWorker{CheckUrl: checkUrl, Redis: redis, CheckQueue: checkQueue, MaxBodySize: maxBodySize}
		workers[i] = worker
	}
	return AnonyChecker{CheckUrl: checkUrl, Redis: redis, CheckQueue: checkQueue, Workers: workers}
}

func (c AnonyChecker) CheckAll() {
	for _, worker := range c.Workers {
		go worker.DoWork()
	}
	for {
		proxy, err := c.pullForCheck()
		if err != nil {
			glog.Errorln("pull for anonymous check error: ", err)
			time.Sleep(5 * time.Second)
			continue
		}
		c.CheckQueue <- proxy
	}
}

func (c AnonyChecker) pullForCheck() (core.Proxy, error) {
	jsonText, err := c.Redis.Lpop(core.PROXY_CHECK_QUEUE)
	if err != nil {
		return core.Proxy{}, err
	}
	checkProxy := core.Proxy{}
	err = json.Unmarshal([]byte(jsonText), &checkProxy)
	return checkProxy, err
}

func (w AnonyCheckWorker) DoWork() {
	for {
		checkProxy := <-w.CheckQueue
		glog.Errorln("anony checker: ", checkProxy)
		proxyStr := checkProxy.Ip + ":" + fmt.Sprintf("%d", checkProxy.Port)
		isAnnoy := httputil.GetForCheck(w.CheckUrl, proxyStr, "anony", nil, w.MaxBodySize)
		if isAnnoy {
			w.checkSuccess(checkProxy)
		}
	}
}

func (w AnonyCheckWorker) checkSuccess(proxy core.Proxy) {
	glog.Infoln("find anony proxy: ", proxy)
	proxyStr := proxy.Ip + ":" + fmt.Sprintf("%d", proxy.Port)
	w.Redis.Sadd(core.PROXY_POOL_VALID, proxyStr)
	w.Redis.Sadd(core.PROXY_POOL_HISTORY, proxyStr)
	if proxy.Source == core.PROXY_SOURCE_CRAW {
		w.Redis.Incr(core.GetProxyTimeKey(core.PROXY_COUNT_CRAW))
	} else if proxy.Source == core.PROXY_SOURCE_SCAN {
		w.Redis.Incr(core.GetProxyTimeKey(core.PROXY_COUNT_SCAN))
	}
}
