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
	crawler.crawlerLogger=*log.New(crawler.createFile(config.log), "[CRAWLER] ", log.Lshortfile)
	crawler.createFile(config.output)
	crawler.visited=mapset.NewSet[string]()
	crawler.fetchClient.Init(config)
	crawler.atomicCounter.Store(0)
	crawler.amountOfGorutines=runtime.NumCPU()*3
	crawler.config=*config
}

func (crawler* CliCrawler) Crawle() ([]byte,error){
	var result []ResourseNode
	ctx, cancel := context.WithTimeout(context.Background(), crawler.config.timeout)
	var wg sync.WaitGroup
	defer cancel()

	resourseChans:=make(chan ResourseNode,10)
	for i:=0;i<len(crawler.config.urls);i++ {
		wg.Add(1)
		domain:=crawler.config.urls[i].ExtractDomain()
		if(crawler.atomicCounter.Load()<int32(crawler.amountOfGorutines)){
			crawler.atomicCounter.Add(1)
			go	crawler.crawleInside(0,ctx,crawler.config.urls[i],resourseChans,&wg,&domain)
		}else{
			crawler.crawlerLogger.Println("Reach gorutines limit:"+crawler.config.urls[i].adress)
			crawler.crawleInside(0,ctx,crawler.config.urls[i],resourseChans,&wg,&domain)
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
			case resource:=<-resourseChans:
		 		result= append(result , resource)
			case <-ctx.Done():
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
			case resource:=<-resourseChans:
				resourceNode.Links = append(resourceNode.Links , resource)
			case <-ctx.Done():
				break loop
		}	
	}
	resources<-resourceNode
}


func ( crawler* CliCrawler) parsePage(page* string,domainUrl* string) ResourseNode{
	var resorce ResourseNode
	reTitle:=regexp.MustCompile("<title>.*</title>")
	title:=reTitle.FindString(*page)
	resorce.Resourse=title[7:len(title)-8]
	reHref := regexp.MustCompile(`href=[\"']([^\"']+)[\"']`)
	hrefs:=reHref.FindAll([]byte(*page),-1)
	crawler.crawlerLogger.Println(2,"Found links on page "+resorce.Resourse+":"+strconv.Itoa(len(hrefs)))
	var links []ResourseNode
	for _,href:= range hrefs{
		crawler.crawlerLogger.Println(2,resorce.Resourse+":"+string(href))
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

