package clicrawler

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)
type FetchClient struct{
	client http.Client // потокобезопасен
	errStatusLogger log.Logger // базовая реализация логгера потокобезопасна
}
func (client* FetchClient) Init(config*CrawlerConfig){
	client.client=http.Client{
    	Timeout: config.requestTimeout,
	}
	file, _ := os.OpenFile(client.extractErrStatusLogFilepath(config.log), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	client.errStatusLogger=*log.New(file, "[FETCH CLIENT] ", log.Lshortfile)
}
func (client* FetchClient) extractErrStatusLogFilepath(logFilepath string) string{
	i:=len(logFilepath)-1
	for ; i> -1&&logFilepath[i]!='.';i--{}
	return logFilepath[:i]+"_error."+logFilepath[i+1:]
}
//планировщик go самостоятельно распределит ресурсы между корутинами, поэтому тут не вижу смысла параллелить
func(client* FetchClient) FetchPage(ctx context.Context,url string,crawler *CliCrawler) string{
	retryAmount:=0
	for{
		select{
			case <-ctx.Done():
			crawler.crawlerLogger.Println("Gracefully stopping fetching")
			return ""
			default:
				if(retryAmount<crawler.config.retry){
					resp, err := client.client.Get(url)
					if err!=nil{
						client.errStatusLogger.Println("Request status code:"+resp.Status)
						client.errStatusLogger.Println("Error while get request:"+err.Error())
						retryAmount++
						select {
							case <-time.After(crawler.config.delay):
								crawler.crawlerLogger.Println("Gorutine try's again after delay:"+url)
		
							case <-ctx.Done():
								return ""
							}
					}
					defer resp.Body.Close()
					client.errStatusLogger.Println("Request status code:"+resp.Status)
					body,err:=io.ReadAll(resp.Body)
					if(err!=nil){
						crawler.crawlerLogger.Println("Error while reading body:"+err.Error())
					}
					crawler.visited.Add(url)
					return string(body)
				}else{
					crawler.crawlerLogger.Println("Can not reach resource:"+url)
					crawler.visited.Add(url)
					return ""
				}
		}
	}		
	
}