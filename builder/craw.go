package builder

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	core "fproxy/core"
	store "fproxy/store"
	"github.com/golang/glog"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CrawTask struct {
	Url       string
	UserAgent string
	Template  CrawTemplate
	WaitTime  int
	Level     int
	MaxLevel  int
}

type Crawler interface {
	Craw()
}

type SimpleCrawler struct {
	UserAgent string
	Tasks     []CrawTask
	Random    *rand.Rand
	Redis     *store.RedisManager
	Distance  int
}

func NewSimpleCrawler(userAgent string, tasks []CrawTask, redis *store.RedisManager, distance int) *SimpleCrawler {
	source := rand.NewSource(rand.Int63())
	random := rand.New(source)
	return &SimpleCrawler{UserAgent: userAgent, Tasks: tasks, Random: random, Redis: redis, Distance: distance}
}

func (c *SimpleCrawler) Craw() {
	tasks := c.Tasks
	for _, task := range tasks {
		c.crawTask(task)
	}
}

func (c *SimpleCrawler) crawTask(task CrawTask) {
	if task.Level >= task.MaxLevel {
		return
	}
	waitTime := time.Duration(c.Random.Intn(task.WaitTime)) * time.Second
	time.Sleep(waitTime)
	html, err := c.downloadHtml(task)
	if err != nil {
		return
	}
	glog.Infoln("template for process: ", task.Template)
	crawResults, err := ProcessCrawTemplate(html, task.Template)
	if err != nil {
		glog.Errorln("craw task process template {"+task.Url+"} error: ", err)
		return
	}
	c.processCrawResults(task, crawResults)
}

func (c *SimpleCrawler) downloadHtml(task CrawTask) (string, error) {
	client := &http.Client{}
	httpRequest, err := http.NewRequest("GET", task.Url, nil)
	if err != nil {
		glog.Errorln("craw task new request{"+task.Url+"} error: ", err)
		return "", err
	}
	userAgent := task.UserAgent
	if userAgent == "" {
		userAgent = c.UserAgent
	}
	httpRequest.Header.Add("User-Agent", userAgent)
	httpResponse, err := client.Do(httpRequest)
	if err != nil {
		glog.Errorln("craw task new request{"+task.Url+"} error: ", err)
		return "", err
	}
	body := httpResponse.Body
	defer body.Close()
	html, err := ioutil.ReadAll(body)
	if err != nil {
		glog.Errorln("craw task read http response{"+task.Url+"} error: ", err)
		return "", err
	}
	return string(html), nil
}

func (s *SimpleCrawler) processCrawResults(srcTask CrawTask, crawResults []CrawResult) {
	crawResultMap := mapCrawResults(crawResults)
	ipAndPortsResult, ok := crawResultMap["ip_port"]
	if ok {
		ipArr := s.processIPWithPorts(ipAndPortsResult)
		if ipArr != nil {
			s.createAndPushIPSections(ipArr)
		}
	} else {
		ipResult, ok1 := crawResultMap["ip"]
		portResult, ok2 := crawResultMap["port"]
		if ok1 && ok2 {
			ipArr := s.processIPAndPorts(ipResult, portResult)
			if ipArr != nil {
				s.createAndPushIPSections(ipArr)
			}
		}
	}
	pageResult, ok := crawResultMap["page"]
	if ok {
		s.processPages(srcTask, pageResult)
	}
}

func mapCrawResults(crawResults []CrawResult) map[string]CrawResult {
	resultMap := make(map[string]CrawResult)
	for _, crawResult := range crawResults {
		if crawResult.Type == CRAW_RESULT_TYPE_EMPTY {
			continue
		}
		resultMap[crawResult.Name] = crawResult
	}
	return resultMap
}

func (s *SimpleCrawler) processIPWithPorts(crawResult CrawResult) []string {
	if crawResult.Value == "" {
		return nil
	}
	ipPortsArr := strings.Split(crawResult.Value, ",")
	ipslen := len(ipPortsArr)
	if ipslen == 0 {
		return nil
	}
	ipArr := make([]string, ipslen)
	for i, ipPortStr := range ipPortsArr {
		ipPortArr := strings.Split(ipPortStr, ":")
		ip := ipPortArr[0]
		port, err := strconv.Atoi(ipPortArr[1])
		if err != nil {
			glog.Errorln("process ip with port error[", ipPortStr, "]: ", err)
			continue
		}
		proxy := core.Proxy{Ip: ip, Port: port, Source: core.PROXY_SOURCE_CRAW}
		s.pushProxyForCheck(proxy)
		ipArr[i] = ip
	}
	return ipArr
}

func (s *SimpleCrawler) processIPAndPorts(ipResult CrawResult, portResult CrawResult) []string {
	if ipResult.Value == "" {
		return nil
	}
	ipsArr := strings.Split(ipResult.Value, ",")
	portsArr := strings.Split(portResult.Value, ",")
	for i, ip := range ipsArr {
		port, err := strconv.Atoi(portsArr[i])
		glog.Infoln("process ip and port: ", ip, ", ", port)
		if err != nil {
			glog.Errorln("process ip and port error[", ip, ":", portsArr[i], "]: ", err)
			continue
		}
		proxy := core.Proxy{Ip: ip, Port: port, Source: core.PROXY_SOURCE_CRAW}
		s.pushProxyForCheck(proxy)
	}
	return ipsArr
}

func (s *SimpleCrawler) processPages(srcTask CrawTask, pageResult CrawResult) {
	if pageResult.Value == "" {
		return
	}
	urlsArr := strings.Split(pageResult.Value, ",")
	for _, url := range urlsArr {
		newUrl := url
		if !strings.HasPrefix(url, "http") {

		}
		newTask := CrawTask{Url: newUrl, UserAgent: srcTask.UserAgent, Template: srcTask.Template, WaitTime: srcTask.WaitTime, Level: srcTask.Level + 1}
		s.crawTask(newTask)
	}
}

func (s *SimpleCrawler) pushProxyForCheck(proxy core.Proxy) {
	jsonBytes, err := json.Marshal(proxy)
	if err != nil {
		glog.Errorln("push craw proxy ", proxy, " for check parse json error: ", err)
		return
	}
	s.Redis.Rpush(core.PROXY_CHECK_QUEUE, string(jsonBytes))
}

func (s *SimpleCrawler) createAndPushIPSections(ipArr []string) {
	ipSections := createIPSections(ipArr, s.Distance)
	for _, ipSection := range ipSections {
		bs, err := json.Marshal(ipSection)
		if err != nil {
			glog.Errorln("marshal craw ip section[", ipSection, "] error: ", err)
			continue
		}
		s.Redis.Rpush(KEY_SCAN_TASK, string(bs))
	}
}

func createIPSections(ipArr []string, distance int) []IPSection {
	sort.Strings(ipArr)
	arrlen := len(ipArr)
	stack := make([]string, arrlen)
	index := 0
	for i := 0; i < arrlen; i++ {
		ip1 := ipArr[i]
		ip1Parts := strings.Split(ip1, ".")
		ip1Parts[3] = "0"
		newIPSec := strings.Join(ip1Parts, ".")
		if index == 0 {
			ipSecs := newIPSec + "_" + newIPSec
			stack[index] = ipSecs
			index++
		} else {
			ipSecs := stack[index-1]
			ipSecArr := strings.Split(ipSecs, "_")
			startSecParts := strings.Split(ipSecArr[0], ".")
			endSecParts := strings.Split(ipSecArr[1], ".")
			isMerge := false
			if ip1Parts[0] == startSecParts[0] && ip1Parts[1] == startSecParts[1] {
				oldStartC, _ := strconv.Atoi(startSecParts[2])
				oldEndC, _ := strconv.Atoi(endSecParts[2])
				newC, _ := strconv.Atoi(ip1Parts[2])
				if newC < oldStartC && (oldStartC-newC) <= distance {
					startSecParts[2] = fmt.Sprintf("%d", newC)
					isMerge = true
				} else if newC > oldEndC && (newC-oldEndC) <= distance {
					endSecParts[2] = fmt.Sprintf("%d", newC)
					isMerge = true
				} else if newC >= oldStartC && newC <= oldEndC {
					isMerge = true
				}
			}
			if isMerge {
				stack[index-1] = strings.Join(startSecParts, ".") + "_" + strings.Join(endSecParts, ".")
			} else {
				ipSecs := newIPSec + "_" + newIPSec
				stack[index] = ipSecs
				index++
			}
		}
	}
	ipSections := make([]IPSection, index)
	for i := 0; i < index; i++ {
		ipSecStr := stack[i]
		ipSecArr := strings.Split(ipSecStr, "_")
		ipSection := IPSection{Start: ipSecArr[0], End: ipSecArr[1], ProxyNum: -1}
		ipSections[i] = ipSection
	}
	return ipSections
}

type CrawTasksXml struct {
	XMLName xml.Name      `xml:"tasks"`
	Tasks   []CrawTaskXml `xml:"task"`
}

type CrawTaskXml struct {
	XMLName   xml.Name `xml:"task"`
	Url       string   `xml:"url,attr"`
	UserAgent string   `xml:"useragent,attr"`
	Template  string   `xml:"template,attr"`
	WaitTime  int      `xml:"wait,attr"`
	MaxLevel  int      `xml:"maxlevel,attr"`
}

type CrawTemplateXml struct {
	XMLName xml.Name       `xml:"template"`
	Name    string         `xml:"name,attr"`
	Entries []CrawEntryXml `xml:"entry"`
}

type CrawEntryXml struct {
	XMLName xml.Name `xml:"entry"`
	Name    string   `xml:"name,attr"`
	Type    string   `xml:"type,attr"`
	Regex   string   `xml:"regex"`
	Xpath   string   `xml:"xpath"`
	Value   string   `xml:"value"`
	MaxPage int      `xml:"maxpage,attr"`
}

func ParseTasksXML(xmlpath, templatedir string) ([]CrawTask, error) {
	content, err := ioutil.ReadFile(xmlpath)
	if err != nil {
		glog.Errorln("parse task xml[", xmlpath, "] error: ", err)
		return nil, errors.New("parse task xml error")
	}
	var tasksXml CrawTasksXml
	err = xml.Unmarshal(content, &tasksXml)
	if err != nil {
		glog.Errorln("unmarshal task xml[", xmlpath, "] error: ", err)
		return nil, errors.New("unmarshal task xml error")
	}
	taskXmls := tasksXml.Tasks
	xmllen := len(taskXmls)
	tasks := make([]CrawTask, xmllen)
	templateMap := make(map[string]*CrawTemplate)
	for i, taskXml := range taskXmls {
		template, err := getTemplate(templateMap, templatedir, taskXml.Template)
		if err != nil {
			return nil, err
		}
		task := CrawTask{Url: taskXml.Url, UserAgent: taskXml.UserAgent, Template: *template, WaitTime: taskXml.WaitTime, Level: 0, MaxLevel: taskXml.MaxLevel}
		tasks[i] = task
	}
	return tasks, nil
}

func getTemplate(templateMap map[string]*CrawTemplate, templatedir, templatename string) (*CrawTemplate, error) {
	template := templateMap[templatename]
	if template != nil {
		return template, nil
	}
	templatepath := templatedir + "/" + templatename + ".xml"
	content, err := ioutil.ReadFile(templatepath)
	if err != nil {
		glog.Errorln("read template[", templatepath, "] error", err)
		return nil, errors.New("read template error")
	}
	newTemplateXml := CrawTemplateXml{}
	err = xml.Unmarshal(content, &newTemplateXml)
	if err != nil {
		glog.Errorln("unmarshal template[", templatepath, "] error", err)
		return nil, errors.New("unmarshal template error")
	}
	entrylen := len(newTemplateXml.Entries)
	entries := make([]CrawEntry, entrylen)
	entryXmls := newTemplateXml.Entries
	for i, entryXml := range entryXmls {
		crawEntry := CrawEntry{Name: entryXml.Name, Type: entryXml.Type, Regex: strings.TrimSpace(entryXml.Regex), Xpath: strings.TrimSpace(entryXml.Xpath), Value: strings.TrimSpace(entryXml.Value), MaxPage: entryXml.MaxPage}
		entries[i] = crawEntry
	}
	newTemplate := CrawTemplate{Name: newTemplateXml.Name, Entries: entries}
	return &newTemplate, nil
}
