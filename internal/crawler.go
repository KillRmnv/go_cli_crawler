package clicrawler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"github.com/deckarep/golang-set/v2"
)

type CliCrawler struct{
	config CrawlerConfig // только чтение, поэтому также безопасен
	visited mapset.Set[string] // потокобезопасная реализация множества
	crawlerLogger log.Logger // базовая реализация логгера потокобезопасна
	fetchClient FetchClient // потокоьезопасен
	atomicCounter atomic.Int32
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
	crawler.atomicCounter.Store(0)
	crawler.amountOfGorutines=runtime.NumCPU()*2
	crawler.parser=&StandardHTMLParser{logger: *log.New(logFile, "[PARSER] ", log.LstdFlags|log.Lshortfile),iterateStubs: config.stubs}
	crawler.storage=&JSONStorage{filepath: crawler.config.output,logger:*log.New(logFile, "[STORAGE] ", log.LstdFlags|log.Lshortfile) }
}

func (crawler* CliCrawler) Crawle() ([]byte,error){
	var result []ResourseNode
	
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
		for i:=0;i<len(links);i++ {
			seedURL,seedDomain:=links[i],&domains[i]
			if(!crawler.visited.Add(seedURL.adress)){
				crawler.crawlerLogger.Println("Skip already visited seed: "+seedURL.adress)
				continue
			}
			wg.Add(1)
			if(crawler.atomicCounter.Load()<int32(crawler.amountOfGorutines)){
				crawler.atomicCounter.Add(1)
				go func(){
					defer crawler.atomicCounter.Add(-1)
					crawler.crawleInside(0,ctx,seedURL,resourseChans,&wg,seedDomain)
				}()
			}else{
				crawler.crawlerLogger.Println("Reach gorutines limit:"+seedURL.adress)
				crawler.crawleInside(0,ctx,seedURL,resourseChans,&wg,seedDomain)
			}
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

// depth должна копироваться, ctx интерфейс, поэтому по дефолту ссылка
func (crawler* CliCrawler) crawleInside(depth int,ctx context.Context,url Url, resources chan ResourseNode,wgParent* sync.WaitGroup,domain*string) { 
	defer wgParent.Done()
	if(depth>crawler.config.depth){
		return
	}
	page:=crawler.fetchClient.FetchPage(ctx,url.adress,crawler)
	if len(page)==0{
		return
	}
	resourceNode,hrefsToCrawl:=crawler.parser.ParsePage(&page,domain,&url.adress)
	pageUrl:=resourceNode.Resourse
	resourseChans:=make(chan ResourseNode,crawler.amountOfGorutines)
	var wg sync.WaitGroup

	go func() {
		defer close(resourseChans)
		for _,href:=range hrefsToCrawl{
			if(ctx.Err()!=nil){
				break
			}
			if(!crawler.visited.Add(href)){
				crawler.crawlerLogger.Println("["+pageUrl+"]"+"Found already visited link "+href)
				continue
			}
			crawler.crawlerLogger.Println("["+pageUrl+"]"+"Crawling to "+href)
			wg.Add(1)
			if(crawler.atomicCounter.Load()<int32(crawler.amountOfGorutines)){
				crawler.atomicCounter.Add(1)
				go func(u Url){
					defer crawler.atomicCounter.Add(-1)
					crawler.crawleInside(depth+1,ctx,u,resourseChans,&wg,domain)
				}(NewUrl(href))
			}else{
				crawler.crawleInside(depth+1,ctx,NewUrl(href),resourseChans,&wg,domain)
			}
		}
		wg.Wait()
		crawler.crawlerLogger.Println("Closing chans for "+url.adress)
	}()

	for resource:=range resourseChans{
		resourceNode.Links=append(resourceNode.Links,resource)
	}

	if !crawler.sendNode(resources,resourceNode,ctx){
		crawler.crawlerLogger.Println("Node dropped on cancel: "+url.adress)
	}
}

func (crawler* CliCrawler) sendNode(resources chan ResourseNode, node ResourseNode, ctx context.Context) bool{
	select{
		case resources<-node:
			return true
		default:
			select{
				case resources<-node:
					return true
				case <-ctx.Done():
					return false
			}
	}
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
