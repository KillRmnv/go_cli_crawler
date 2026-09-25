package clicrawler

import "time"
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
func(config*CrawlerConfig) SetDepth(depth string ){
	
}
func(config*CrawlerConfig) SetTimeout(timeoutStr string ){
	
}
func(config*CrawlerConfig) SetRequestTimeout( requestTimeoutStr string ){
	
}
func(config*CrawlerConfig) SetOutput( outputPath string ){
	
}
func(config*CrawlerConfig) SetLog( logPath string ){
	
}