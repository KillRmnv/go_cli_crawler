package main

import (
	"crawler/internal"
	"flag"
	"time"
)
const(
	depth=10
	timeout =10
	requestTimeout=2
	outputFilepath="./out/result.json"
	logFilepath="./log/crawler.log"
	retry=1
	delay=0
)

func main() {

	urls := flag.String("urls", "", "a urls to crawl")
	depth := flag.Int("depth", depth, "a recursion depth")
	timeout := flag.Duration("timeout", time.Second*timeout, "an overall timeout for application")
	reqTimeout := flag.Duration("request-timeout", time.Minute*requestTimeout, "a timeout for a single request")
	output := flag.String("output", outputFilepath, "an output json filepath")
	logPath := flag.String("log", logFilepath, "a log filepath")
	retry:=flag.Int("retry",retry,"amount of request retries")
	delay:=flag.Duration("delay",delay,"time between to requests")
	
	flag.Parse() 
	
	config := clicrawler.CrawlerConfig{}
	config.SetUrls(*urls)
	config.SetDepth(*depth)
	config.SetTimeout(*timeout)
	config.SetRequestTimeout(*reqTimeout)
	config.SetOutput(*output)
	config.SetLog(*logPath)
	config.SetRetry(*retry)
	config.SetDelay(*delay)
	var crawler clicrawler.CliCrawler
	crawler.Init(config)
	crawler.Crawle()
}