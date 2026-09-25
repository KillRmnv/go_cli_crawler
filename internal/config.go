package clicrawler

import (
	"time"
)

type CrawlerConfig struct{
	urls[] string
	depth int
	timeout time.Duration
	requestTimeout time.Duration
	output string
	log string
	
}
func(config*CrawlerConfig) SetUrls(substrOfParams string ){
	
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