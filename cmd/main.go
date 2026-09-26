package main

import (
	"crawler/internal"
	"flag"
)

func main() {

	urls := flag.String("urls", "", "a urls to crawl")
	depth := flag.Int("depth", clicrawler.Depth, "a recursion depth")
	timeout := flag.Duration("timeout", clicrawler.Timeout, "an overall timeout for application")
	reqTimeout := flag.Duration("request-timeout", clicrawler.RequestTimeout, "a timeout for a single request")
	output := flag.String("output", clicrawler.OutputFilepath, "an output json filepath")
	logPath := flag.String("log", clicrawler.LogFilepath, "a log filepath")
	retry:=flag.Int("retry",clicrawler.Retry,"amount of request retries")
	delay:=flag.Duration("delay",clicrawler.Delay,"time between to requests")
	
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
	crawler.Init(&config)
	crawler.Crawle()
}