package clicrawler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"github.com/deckarep/golang-set/v2"
)
const maxConcurrentRequests = 10
type CliCrawler struct{
	config CrawlerConfig // только чтение, поэтому также безопасен
	visited mapset.Set[string] // потокобезопасная реализация множества
	crawlerLogger log.Logger // базовая реализация логгера потокобезопасна
	fetchClient FetchClient // потокоьезопасен
	semaphore chan struct{}
	amountOfGorutines int // потокобезопасно: инициализируется и читается
	parser Parser
	storage Storage
}

func (crawler* CliCrawler) createFile(filePath string) *os.File{
	filePath=strings.Trim(filePath," ")
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal("Can not create directory for file:"+ err.Error())
	}
	checkFile,err := os.OpenFile(filePath,os.O_RDONLY, 0666)
	if err==nil{
		checkFile.Close()
		now:=time.Now()
		ext:=filepath.Ext(filePath)
		if ext!=""{
			filePath= filePath[:len(filePath)-len(ext)]+now.Format("2006-01-02_15_04")+ext
		}else{
			filePath= filePath+now.Format("2006-01-02_15_04")+".json"
		}
	}
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
			log.Fatal("Can not open file"+ err.Error())
	}
	return file
}

func (crawler* CliCrawler) Init(config* CrawlerConfig){
	crawler.config=*config
	logFile:= crawler.createFile(crawler.config.log)
	crawler.crawlerLogger=*log.New(logFile, "[CRAWLER] ", log.LstdFlags|log.Lshortfile)
	outFile:=crawler.createFile(crawler.config.output) 
	crawler.config.SetOutput(outFile.Name())
	outFile.Close()
	crawler.visited=mapset.NewSet[string]()
	crawler.fetchClient.Init(config)
	crawler.amountOfGorutines=maxConcurrentRequests
	crawler.semaphore = make(chan struct{}, crawler.amountOfGorutines)
	crawler.parser=&StandardHTMLParser{logger: *log.New(logFile, "[PARSER] ", log.LstdFlags|log.Lshortfile),iterateStubs: config.stubs}
	crawler.storage=&JSONStorage{filepath: crawler.config.output,logger:*log.New(logFile, "[STORAGE] ", log.LstdFlags|log.Lshortfile) }
}

func (crawler* CliCrawler) Crawle() ([]byte,error){
	var result []ResourseNode
	crawler.crawlerLogger.Printf("Config:%v",crawler.config)
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, crawler.config.timeout)
	defer cancel()
	go func() {
		<-ctx.Done()
		stop() 
	}()
	links,domains:=crawler.extractUrlsDomains()
	resourseChans:=make(chan ResourseNode,crawler.amountOfGorutines)
	var wg sync.WaitGroup
	
	go func() {
    defer close(resourseChans)
    for i := 0; i < len(links); i++ {
        seedURL, seedDomain := links[i], &domains[i]
        if !crawler.visited.Add(seedURL.adress) {
            crawler.crawlerLogger.Println("Skip already visited seed: "+seedURL.adress)
            continue
        }
        wg.Add(1)
			go crawler.crawleInside(0, ctx, seedURL, resourseChans, &wg, seedDomain)
		}
    wg.Wait()
    crawler.crawlerLogger.Println("Closing chans for root")
	}()
	for resource:=range resourseChans{
		result=append(result,resource)
	}
	data,saveErr:=crawler.storage.Save(result)
	if saveErr!=nil{
		return data,saveErr
	}
	
	if len(result)==0{
		if len(links)==0{
			return data,errors.New("no valid urls to crawl")
		}
		if ctx.Err()==nil{
			return data,fmt.Errorf("nothing fetched: all %d seed fetches failed",len(links))
		}
	}
	return data,nil
}

func (crawler *CliCrawler) acquire(ctx context.Context) bool {
	select {
	case crawler.semaphore <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (crawler* CliCrawler) crawleInside(depth int, ctx context.Context, url Url, resources chan ResourseNode, wgParent* sync.WaitGroup, domain*string) { 
    defer wgParent.Done()
	if !crawler.acquire(ctx) {
		return
	}
	page := crawler.fetchClient.FetchPage(ctx, url.adress, crawler)
	var resourceNode ResourseNode
	var hrefsToCrawl []string
	if len(page) > 0 {
		resourceNode, hrefsToCrawl = crawler.parser.ParsePage(&page, domain, &url.adress)
	}
	<-crawler.semaphore

	if len(page) == 0 {
        return 
    }
	if resourceNode.Links == nil {
		resourceNode.Links = []ResourseNode{}
	}

	if depth < crawler.config.depth {
        pageUrl := resourceNode.Resourse
        resourseChans := make(chan ResourseNode, crawler.amountOfGorutines)
        var wg sync.WaitGroup
    
        go func() {
            defer close(resourseChans)
            for _, href := range hrefsToCrawl {
                if ctx.Err() != nil {
                    break
                }
                if !crawler.visited.Add(href) {
                    crawler.crawlerLogger.Println("["+pageUrl+"]"+"Found already visited link "+href)
                    continue
                }
                
                crawler.crawlerLogger.Println("["+pageUrl+"]"+"Crawling to "+href)
                wg.Add(1)
				go crawler.crawleInside(depth+1, ctx, NewUrl(href), resourseChans, &wg, domain)
			}
			wg.Wait()
            crawler.crawlerLogger.Println("Closing chans for "+url.adress)
        }()
        for resource := range resourseChans {
            resourceNode.Links = append(resourceNode.Links, resource)
        }
    }

	resources <- resourceNode
}

func (crawler* CliCrawler) extractUrlsDomains() ([]Url,[]string){
	var links []Url
	var domains []string
	for _,u := range crawler.config.urls{
		domain,err:=u.ExtractDomain()
		if(err!=nil){
			crawler.crawlerLogger.Println("Invalid url: "+u.adress+" ("+err.Error()+")")
			continue
		}
		crawler.crawlerLogger.Printf("Extracted domain:%v for %v\n",domain,u)
		links=append(links,u)
		domains=append(domains,domain)
	}
	return links, domains
}
