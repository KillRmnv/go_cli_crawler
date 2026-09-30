package clicrawler

import (
	"strings"
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
	if delayTimeout<0{
		delayTimeout=Delay
	}
	config.delay=delayTimeout
}
func(config*CrawlerConfig) SetRetry(amount int ){
	if amount<1{
		amount=Retry
	}
	config.retry=amount
}
func (config*CrawlerConfig) SetUrlsSlice(slice []string){
	for _,v:=range slice{
		if trimmed:=strings.TrimSpace(v); trimmed!=""{
			config.urls=append(config.urls, NewUrl(trimmed))
		}
	}
}
func(config*CrawlerConfig) SetUrls(substrOfParams string ){
	for _,v:=range strings.Split(substrOfParams,","){
		if trimmed:=strings.TrimSpace(v); trimmed!=""{
			config.urls=append(config.urls, NewUrl(trimmed))
		}
	}
}
func(config*CrawlerConfig) SetDepth(depth int ){
	if depth<0{
		depth=Depth
	}
	config.depth=depth
}
func(config*CrawlerConfig) SetTimeout(timeout time.Duration ){
	if timeout<=0{
		timeout=Timeout
	}
	config.timeout=timeout
}
func(config*CrawlerConfig) SetRequestTimeout( requestTimeout time.Duration ){
	if requestTimeout<=0{
		requestTimeout=RequestTimeout
	}
	config.requestTimeout=requestTimeout
}
func(config*CrawlerConfig) SetOutput( outputPath string ){
	if strings.TrimSpace(outputPath)==""{
		outputPath=OutputFilepath
	}
	config.output=strings.TrimSpace(outputPath)
}
func(config*CrawlerConfig) SetLog( logPath string ){
	if strings.TrimSpace(logPath)==""{
		logPath=LogFilepath
	}
	config.log=strings.TrimSpace(logPath)
}