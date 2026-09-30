package clicrawler

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)
type FetchClient struct{
	client http.Client // потокобезопасен
	errStatusLogger log.Logger // базовая реализация логгера потокобезопасна
	semaphore       chan struct{}
}
func (client* FetchClient) Init(config*CrawlerConfig){
	client.client=http.Client{
    	Timeout: config.requestTimeout,
    	CheckRedirect: func(req *http.Request, via []*http.Request) error {
             return http.ErrUseLastResponse
         },
	}
	logErrFilepath:=client.extractErrStatusLogFilepath(config.log)
	dir := filepath.Dir(logErrFilepath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal("Can not create directory for logs:"+ err.Error())
	}
	file, err := os.OpenFile(logErrFilepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
			log.Println("Can not open log file"+ err.Error())
	}
	client.errStatusLogger=*log.New(file, "[FETCH CLIENT] ", log.LstdFlags|log.Lshortfile)
	client.semaphore = make(chan struct{}, 10)
}

func (client* FetchClient) extractErrStatusLogFilepath(logFilepath string) string{
	logFilepath=strings.Trim(logFilepath," ")
	ext:=filepath.Ext(logFilepath)
	if ext!=""{
		return logFilepath[:len(logFilepath)-len(ext)]+"_error"+ext
	}
	return logFilepath+"_error.log"
}
//планировщик go самостоятельно распределит ресурсы между корутинами, поэтому тут не вижу смысла параллелить
func(client* FetchClient) FetchPage(ctx context.Context,url string,crawler *CliCrawler) string{
	if ctx.Err()!=nil{
		return ""
	}
	crawler.crawlerLogger.Println("Trying to fetch:"+url)
	select {
		case client.semaphore <- struct{}{}: 
			defer func() { <-client.semaphore }()
		case <-ctx.Done():
			crawler.crawlerLogger.Println("Gracefully stopping fetching (in queue)")
			return ""
	}
	retryAmount,isOkFormat:=0,false
	for{
		select{	
			case <-ctx.Done():
				crawler.crawlerLogger.Println("Gracefully stopping fetching")
				return ""
			default:
				if !isOkFormat{
					if(retryAmount<crawler.config.retry){
						isHtml,isContinue:=client.processHead(ctx,url,crawler,&retryAmount)
						crawler.visited.Add(url)
						if isContinue{
							continue
						}
						if !isHtml {
							return ""
						}		
					}
					isOkFormat=true
					retryAmount=0
				}
				if(retryAmount<crawler.config.retry){
					result,flag:=client.processGet(ctx,url,crawler,&retryAmount)
					if !flag{
						return result
					}
				}else{
					crawler.crawlerLogger.Println("Can not reach resource:"+url)
					crawler.visited.Add(url)
					return ""
				}
		}
	}		
}

func(client* FetchClient) processGet(ctx context.Context,url string,crawler *CliCrawler, retryAmount* int) (string,bool){
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := client.client.Do(req)
	if err!=nil{
		client.errStatusLogger.Printf("Error while get request:%s",err.Error())
		*retryAmount++
		select {
			case <-time.After(crawler.config.delay):
				crawler.crawlerLogger.Println("Gorutine try's again after delay:"+url)
				return "",true
			case <-ctx.Done():
				crawler.crawlerLogger.Println("Gracefully stopping waiting")
				return "",false
			}
	}
	defer resp.Body.Close()
	crawler.visited.Add(url)			
	client.errStatusLogger.Printf("Request status code:%s; Url:%s",resp.Status,url)
	body,err:=io.ReadAll(resp.Body)
	if(err!=nil){
		crawler.crawlerLogger.Println("Error while reading body:"+err.Error())
	}
	return string(body),false
}

func(client* FetchClient) processHead(ctx context.Context,url string,crawler *CliCrawler, retryAmount* int) (bool,bool){
	req, _ := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	resp, err := client.client.Do(req)
	if err!=nil{
		client.errStatusLogger.Println("Error while get request:"+err.Error())
		*retryAmount++
		select {
			case <-time.After(crawler.config.delay):
				crawler.crawlerLogger.Println("Gorutine try's again after delay:"+url)
				return false,true
			case <-ctx.Done():
				crawler.crawlerLogger.Println("Gracefully stopping waiting")
				return false,false
			}
	}
	defer resp.Body.Close()
	if resp.StatusCode>=400{
		client.errStatusLogger.Printf("Request status code:%s; Url:%s",resp.Status,url)
		return true,false
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
	    crawler.crawlerLogger.Println("Skip non HTML resource:"+url+ " Тип:"+ contentType)
	    return false,false
	}			
	client.errStatusLogger.Printf("Request status code:%s; Url:%s",resp.Status,url)
	return true,false
}
