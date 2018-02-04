package processor

import (
	"fproxy/core"
	"fproxy/store"
	"math/rand"
)

type ChainProcessor struct {
	Processors []Processor
}

func (c *ChainProcessor) Process(proxy core.Proxy) int {
	finalResult := 0
	for _, processor := range c.Processors {
		result := processor.Process(proxy)
		if result == SUCCESS {
			processor.OnSuccess(proxy)
		} else {
			processor.OnFail(proxy)
		}
		finalResult += result
	}
	return finalResult
}

func NewChainProcessor(redisManager *store.RedisManager, requests []CheckRequest) *ChainProcessor {
	random := rand.New(rand.NewSource(rand.Int63()))
	httpProcessor := &HttpProcessor{UserAgent: "", CheckRequests: requests, RedisManager: redisManager, RequestRand: random}
	var processors = []Processor{httpProcessor}
	return &ChainProcessor{Processors: processors}
}
