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
)

func main() {

	config:=clicrawler.CrawlerConfig{}
	config.SetUrls(*flag.String("urls", "", "a urls to crawle"))
	config.SetDepth(*flag.Int("depth",depth,"a recursion depth"))
	config.SetTimeout(*flag.Duration("timeout",time.Second*timeout,"an overall timeout for application"))
	config.SetRequestTimeout(*flag.Duration("request-timeout",time.Minute*requestTimeout,"an overall timeout for application"))
	config.SetOutput(*flag.String("output",outputFilepath,"an output json with result filepath"))
	config.SetLog(*flag.String("log",logFilepath,"a log filepath"))
	flag.Parse()

	
}