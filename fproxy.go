package main

import (
	"flag"
	builder "fproxy/builder"
	"fproxy/builder/processor"
	"fproxy/check"
	"fproxy/config"
	server "fproxy/server"
	store "fproxy/store"
	"github.com/golang/glog"
	"github.com/robfig/cron"
	"time"
)

type CmdArgs struct {
	Conf         string
	Craw         bool
	Scan         bool
	HistoryCheck bool
	AnonyCheck   bool
	Http         bool
}

func main() {
	cmdArgs := readCmd()
	glog.Infoln(cmdArgs)
	config, err := config.ReadConfig(cmdArgs.Conf)
	if err != nil {
		glog.Errorln("read config file error: ", err)
		return
	}
	glog.Infoln("read config complete")
	redis, err := NewRedisManager(config)
	if err != nil {
		glog.Errorln("create redis client error: ", err)
		return
	}
	glog.Infoln("connect redis complete")
	if cmdArgs.Scan {
		scanner, err := NewScanner(config, redis)
		if err != nil {
			glog.Errorln("new scanner error: ", err)
			return
		}
		glog.Infoln("scanner: ", scanner)
		go scanner.Start()
	}
	if cmdArgs.HistoryCheck {
		historyChecker := NewHistoryChecker(config, redis)
		go historyChecker.CheckAll()
	}
	if cmdArgs.AnonyCheck {
		anonyChecker := NewAnonyChecker(config, redis)
		go anonyChecker.CheckAll()
	}
	if cmdArgs.Craw {
		glog.Infoln("create crawler...")
		simpleCrawler, err := NewSimpleCrawler(config, redis)
		if err != nil {
			glog.Errorln("create simple crawler error: ", err)
			return
		}
		croner := cron.New()
		setCrawTask(croner, simpleCrawler)
		croner.Start()
	}
	if cmdArgs.Http {
		server := server.NewFProxyServer()
		server.Init()
	}
	for {
		time.Sleep(10 * time.Second)
	}
}

func readCmd() CmdArgs {
	conf := flag.String("conf", "conf.yml", "配置文件路径")
	craw := flag.Bool("craw", false, "开启爬虫")
	scan := flag.Bool("scan", false, "开启扫描")
	historyCheck := flag.Bool("check-history", false, "开启历史池轮询")
	anonyCheck := flag.Bool("check-anony", false, "开启高匿检测")
	http := flag.Bool("http", false, "开启http服务")
	flag.Parse()
	cmdArgs := CmdArgs{Conf: *conf, Craw: *craw, Scan: *scan, HistoryCheck: *historyCheck, AnonyCheck: *anonyCheck, Http: *http}
	return cmdArgs
}

func NewRedisManager(config config.Config) (*store.RedisManager, error) {
	redisConfig := config.Redis
	timeout := time.Duration(redisConfig.Timeout) * time.Second
	return store.NewRedisManager(redisConfig.Host, redisConfig.Port, redisConfig.Password, redisConfig.Db, redisConfig.MaxIdle, redisConfig.MaxActive, timeout)
}

func NewScanner(config config.Config, redisManager *store.RedisManager) (*builder.Scanner, error) {
	scanConfig := config.Scan
	requests, err := processor.ParseRequestXml(scanConfig.Requests)
	if err != nil {
		return nil, err
	}
	return builder.NewScanner(scanConfig.NWorkers, scanConfig.Ports, redisManager, requests), nil
}

func NewSimpleCrawler(config config.Config, redis *store.RedisManager) (*builder.SimpleCrawler, error) {
	crawConfig := config.Craw
	crawTasks, err := loadCrawTasks(config)
	if err != nil {
		return nil, err
	}
	return builder.NewSimpleCrawler(crawConfig.UserAgent, crawTasks, redis, crawConfig.Distance), nil
}

func loadCrawTasks(config config.Config) ([]builder.CrawTask, error) {
	crawConfig := config.Craw
	taskPath := crawConfig.Task
	templateDir := crawConfig.Template
	tasks, err := builder.ParseTasksXML(taskPath, templateDir)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func setCrawTask(croner *cron.Cron, simpleCrawler *builder.SimpleCrawler) {
	croner.AddFunc("0 0/30 * * * *", func() {
		glog.Infoln("start craw...")
		simpleCrawler.Craw()
	})
}

func NewHistoryChecker(config config.Config, redis *store.RedisManager) *check.HistoryChecker {
	historyConfig := config.Checker.History
	return check.NewHistoryChecker(redis, historyConfig.NWorkers, historyConfig.CheckSize, historyConfig.UserAgent, historyConfig.CheckUrls)
}

func NewAnonyChecker(config config.Config, redis *store.RedisManager) check.AnonyChecker {
	anonyConfig := config.Checker.Anony
	glog.Infoln("anony check config: ", anonyConfig)
	return check.NewAnonyChecker(anonyConfig.CheckUrl, redis, anonyConfig.NWorkers, anonyConfig.CheckSize, anonyConfig.MaxBodySize)
}
