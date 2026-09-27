package clicrawler

import (
	"time"
)
const(
	Depth=10
	Timeout =2*time.Minute
	RequestTimeout=10*time.Second
	OutputFilepath="./out/result.json"
	LogFilepath="./log/crawler.log"
	Retry=1
	Delay=0*time.Second
	Stubs=false
)
type CrawlerConfig struct{
	urls[] Url
	depth int
	timeout time.Duration
	requestTimeout time.Duration
	output string
	log string
	retry int
	delay time.Duration
	stubs bool
	
}
func(config*CrawlerConfig) SetStubs( withStubs bool ){
	config.stubs=withStubs
}
func(config*CrawlerConfig) SetDelay( delayTimeout time.Duration ){
	config.delay=delayTimeout
}
func(config*CrawlerConfig) SetRetry(amount int ){
	config.retry=amount
}
func (config*CrawlerConfig) SetUrlsSlice(slice []string){
	for _,v:=range slice{
		config.urls=append(config.urls, NewUrl(v))
	}
}
func(config*CrawlerConfig) SetUrls(substrOfParams string ){
	start:=0
	for i:=0;i<len(substrOfParams);i++{
		if(substrOfParams[i]==','){
			config.urls=append(config.urls, NewUrl(substrOfParams[start:i]))
			start=i+1
		}
	}
	config.urls=append(config.urls,NewUrl(substrOfParams[start:]))
	
}
func(config*CrawlerConfig) SetDepth(depth int ){
	config.depth=depth
}
func(config*CrawlerConfig) SetTimeout(timeout time.Duration ){
	config.timeout=timeout
}
func(config*CrawlerConfig) SetRequestTimeout( requestTimeout time.Duration ){
	config.requestTimeout=requestTimeout
}
func(config*CrawlerConfig) SetOutput( outputPath string ){
	config.output=outputPath
}
func(config*CrawlerConfig) SetLog( logPath string ){
	config.log=logPath
}