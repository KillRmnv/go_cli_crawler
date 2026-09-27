package clicrawler

import (
	"context"
	"encoding/json/v2"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
}

func (crawler* CliCrawler) createFile(filePath string) *os.File{
	filePath=strings.Trim(filePath," ")
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal("Can not create directory for file:"+ err.Error())
	}
	_,err := os.OpenFile(filePath,os.O_RDONLY, 0666)
	if err==nil{
		i:=len(filePath)-1
		now:=time.Now()
		for ; i> 0&&filePath[i]!='.';i--{}
		if filePath[i]=='.'{
			filePath= filePath[:i]+now.Format("2006-01-02_15_04")+"."+filePath[i+1:]
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
	crawler.crawlerLogger=*log.New(crawler.createFile(crawler.config.log), "[CRAWLER] ", log.Lshortfile)
	outFile:=crawler.createFile(crawler.config.output) 
	crawler.config.SetOutput(outFile.Name())
	outFile.Close()
	crawler.visited=mapset.NewSet[string]()
	crawler.fetchClient.Init(config)
	crawler.atomicCounter.Store(0)
	crawler.amountOfGorutines=runtime.NumCPU()*2
}

func (crawler* CliCrawler) Crawle() ([]byte,error){
	var result []ResourseNode
	
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, crawler.config.timeout)
	defer cancel()

	resourseChans:=make(chan ResourseNode,10)
	for i:=0;i<len(crawler.config.urls);i++ {
		wg.Add(1)
		domain,err:=crawler.config.urls[i].ExtractDomain()
		if(err!=nil){
			crawler.crawlerLogger.Println("Invalid domain:"+crawler.config.urls[i].adress)
			crawler.config.urls=append(crawler.config.urls[:i],crawler.config.urls[i+1:]...)
		}else{
			crawler.crawlerLogger.Printf("Extracted domain:%v for %v\n",domain,crawler.config.urls[i])
			
			if(crawler.atomicCounter.Load()<int32(crawler.amountOfGorutines)){
				crawler.atomicCounter.Add(1)
				go	crawler.crawleInside(0,ctx,crawler.config.urls[i],resourseChans,&wg,&domain)
			}else{
				crawler.crawlerLogger.Println("Reach gorutines limit:"+crawler.config.urls[i].adress)
				crawler.crawleInside(0,ctx,crawler.config.urls[i],resourseChans,&wg,&domain)
			}
		}
	}
	go func() {
			wg.Wait() 
			close(resourseChans) 
			crawler.crawlerLogger.Println("Closing chans for root")
	}()
	loop:
	for{
		select{
			case resource,ok:=<-resourseChans:
				if!ok{
					break loop
				}
		 		result= append(result , resource)
			case <-ctx.Done():
			for resource:=range resourseChans{
				result= append(result , resource)
			}
			break loop
		}	
	}
	json,err:=json.Marshal(result)
	if err!=nil{
		crawler.crawlerLogger.Println("Error occured while serializing")
	}
	os.WriteFile(crawler.config.output,json,os.FileMode(os.O_RDWR))
	return json,nil
}
// depth должна копироваться, ctx интерфейс, поэтому по дефолту ссылка
func (crawler* CliCrawler) crawleInside(depth int,ctx context.Context,url Url, resources chan ResourseNode,wgParent* sync.WaitGroup,domain*string) { 
	defer wgParent.Done()
	defer crawler.atomicCounter.Add(-1)
	depth++
	if(depth>crawler.config.depth){
		crawler.crawlerLogger.Println("Max depth reached:"+strconv.Itoa(depth))
		return
	}
	page:=crawler.fetchClient.FetchPage(ctx,url.adress,crawler)
	if len(page)==0{
		return
	}
	var resourceNode ResourseNode
	resourceNode=crawler.parsePage(&page,domain)
	resourseChans:=make(chan ResourseNode,10)
	var wg sync.WaitGroup
	for _,v:=range resourceNode.Links{
		if(!crawler.visited.Contains(v.Resourse)){
			crawler.crawlerLogger.Println("["+resourceNode.Resourse+"]"+"Crawling to "+v.Resourse)	
			wg.Add(1)
			if(crawler.atomicCounter.Load()<int32(crawler.amountOfGorutines)){
				crawler.atomicCounter.Add(1)
		    	go	crawler.crawleInside(depth,ctx,NewUrl(v.Resourse),resourseChans,&wg,domain)
			}else{
				crawler.crawleInside(depth,ctx,NewUrl(v.Resourse),resourseChans,&wg,domain)
			}
		}else{
			crawler.crawlerLogger.Println("["+resourceNode.Resourse+"]"+"Found already visited link "+v.Resourse)
		}
	}
	go func() {
			wg.Wait() 
			close(resourseChans) 
			crawler.crawlerLogger.Println("Closing chans for "+url.adress)
	}()
	loop:
	for{
		select{
			case resource,ok:=<-resourseChans:
				if !ok{
					break loop
				}
				resourceNode.Links = append(resourceNode.Links , resource)
			case <-ctx.Done():
				for resource:= range resourseChans{
					 resourceNode.Links = append(resourceNode.Links, resource)
				}
				break loop
		}	
	}
	resources<-resourceNode
}


func ( crawler* CliCrawler) parsePage(page* string,domainUrl* string) ResourseNode{
	var resorce ResourseNode
	reTitle:=regexp.MustCompile("<title>.*</title>")
	title:=reTitle.FindString(*page)
	resorce.Title=title[7:len(title)-8]
	resorce.Resourse=*domainUrl
	reHref := regexp.MustCompile(`href=[\"']([^\"']+)[\"']`)
	hrefs:=reHref.FindAll([]byte(*page),-1)
	crawler.crawlerLogger.Println(2,"Found links on page "+resorce.Title+":"+strconv.Itoa(len(hrefs)))
	var links []ResourseNode
	for _,href:= range hrefs{
		crawler.crawlerLogger.Println(2,resorce.Title+":"+string(href))
		hrefParsed:=href[6 : len(href)-1]
		if strings.Contains(string(hrefParsed),*domainUrl){
			links=append(links, ResourseNode{Resourse: string(hrefParsed)})
		}else{
			crawler.crawlerLogger.Println("Href of not parent domain:"+string(href)+" Parent domain:"+*domainUrl)
		}
	}
	resorce.Links=links
	return resorce
}

func ( crawler* CliCrawler) extractUrlDomain() ([]Url,[]string){
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