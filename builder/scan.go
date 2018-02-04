package builder

import (
	"encoding/json"
	"fproxy/builder/processor"
	"fproxy/core"
	"fproxy/store"
	"github.com/golang/glog"
	"strconv"
	"strings"
	"time"
)

const KEY_SCAN_TASK = "proxy:scan:task"

type TaskResult struct {
	IsProxy bool
}

type ProxyTask struct {
	IP   string
	Port int
}

type Worker struct {
	Processor *processor.ChainProcessor
}

func (w *Worker) DoWork(taskChan chan ProxyTask, resultChan chan TaskResult) {
	glog.Infoln("worker start do work...")
	for {
		task := <-taskChan
		proxy := core.Proxy{Ip: task.IP, Port: task.Port, Source: core.PROXY_SOURCE_SCAN}
		proxyNum := w.Processor.Process(proxy)
		isProxy := false
		if proxyNum > 0 {
			isProxy = true
		}
		if isProxy {
			glog.Infoln("scan proxy: ", task.IP, ", ", task.Port)
		}
		taskResult := TaskResult{IsProxy: isProxy}
		resultChan <- taskResult
	}
}

type Scanner struct {
	Ports        []int
	RedisManager *store.RedisManager
	Workers      []*Worker
	TaskChan     chan ProxyTask
	ResultChan   chan TaskResult
}

func NewScanner(nWorkers int, ports []int, redisManager *store.RedisManager, requests []processor.CheckRequest) *Scanner {
	processor := processor.NewChainProcessor(redisManager, requests)
	if nWorkers <= 0 {
		nWorkers = 3
	}
	workers := make([]*Worker, nWorkers)
	for i := 0; i < nWorkers; i++ {
		worker := &Worker{Processor: processor}
		workers[i] = worker
	}
	taskChan := make(chan ProxyTask, 65535)
	resultChan := make(chan TaskResult, 65535)
	return &Scanner{Ports: ports, RedisManager: redisManager, TaskChan: taskChan, ResultChan: resultChan, Workers: workers}
}

func (s *Scanner) Start() {
	glog.Infoln("start proxy scan workers...")
	for _, worker := range s.Workers {
		go worker.DoWork(s.TaskChan, s.ResultChan)
	}
	glog.Infoln("scan workers running, start scanner...")
	for {
		glog.Infoln("pull ipsection for scan...")
		ipSection := s.pullIPSection()
		if ipSection == nil {
			time.Sleep(5 * time.Second)
			continue
		}
		proxyTasks := createProxyTasks(ipSection, s.Ports)
		taskNum := len(proxyTasks)
		glog.Infoln("ip section[", ipSection.Start, ",", ipSection.End, "] task size: ", taskNum)
		for _, task := range proxyTasks {
			s.TaskChan <- task
		}
		proxyNum := 0
		for i := 0; i < taskNum; i++ {
			result := <-s.ResultChan
			if result.IsProxy {
				proxyNum++
			}
		}
		glog.Infoln("ip section[", ipSection.Start, ",", ipSection.End, "] proxyNum size: ", proxyNum)
		if ipSection.ProxyNum == -1 || ipSection.ProxyNum > 0 || proxyNum > 0 {
			ipSection.ProxyNum = proxyNum
			s.pushIPSection(ipSection)
		}
	}
}

func createProxyTasks(ipSection *IPSection, ports []int) []ProxyTask {
	startIP := ipSection.Start
	endIP := ipSection.End
	startIPParts := strings.Split(startIP, ".")
	endIPParts := strings.Split(endIP, ".")
	startCPart, _ := strconv.Atoi(startIPParts[2])
	endCPart, _ := strconv.Atoi(endIPParts[2])
	size := (endCPart - startCPart + 1) * 256 * len(ports)
	proxyTasks := make([]ProxyTask, size)
	index := 0
	for cPart := startCPart; cPart <= endCPart; cPart++ {
		for dPart := 0; dPart < 256; dPart++ {
			ip := startIPParts[0] + "." + startIPParts[1] + "." + strconv.Itoa(cPart) + "." + strconv.Itoa(dPart)
			for _, port := range ports {
				proxyTask := ProxyTask{IP: ip, Port: port}
				proxyTasks[index] = proxyTask
				index++
			}
		}
	}
	return proxyTasks
}

func (s *Scanner) pullIPSection() *IPSection {
	jsonText, err := s.RedisManager.Lpop(KEY_SCAN_TASK)
	if jsonText == "" || err != nil {
		return nil
	}
	ipSection := &IPSection{}
	err = json.Unmarshal([]byte(jsonText), ipSection)
	if err != nil {
		return nil
	}
	return ipSection
}

func (s *Scanner) pushIPSection(ipSection *IPSection) {
	if ipSection == nil {
		return
	}
	bVal, err := json.Marshal(ipSection)
	if err != nil {
		glog.Errorln("marshal ip section error[", ipSection.Start, ", ", ipSection.End, ", ", ipSection.ProxyNum, "]: ", err)
		return
	}
	s.RedisManager.Rpush(KEY_SCAN_TASK, string(bVal))
}
