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
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal("Can not create directory for file:"+ err.Error())
	}
	_,err := os.OpenFile(filePath,os.O_RDONLY, 0666)
	if err==nil{
		now:=time.Now()
		filePath=filePath+now.Format("2006-01-02_15_04")
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
	result:=make([]ResourseNode,10)
	ctx, cancel := context.WithTimeout(context.Background(), crawler.config.timeout)
	var wg sync.WaitGroup
	defer cancel()

	resourseChans:=make(chan ResourseNode,10)
	for i:=0;i<len(crawler.config.urls);i++ {
		wg.Add(1)
		domain:=extractDomain(crawler.config.urls[i])
		if(crawler.atomicCounter.Load()<int32(crawler.amountOfGorutines)){
			crawler.atomicCounter.Add(1)
			go	crawler.crawleInside(0,ctx,crawler.config.urls[i],resourseChans,&wg,&domain)
		}else{
			crawler.crawlerLogger.Println("Reach gorutines limit:"+crawler.config.urls[i])
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
func (crawler* CliCrawler) crawleInside(depth int,ctx context.Context,url string, resources chan ResourseNode,wgParent* sync.WaitGroup,domain*string) { 
	defer wgParent.Done()
	depth++
	if(depth>crawler.config.depth){
		crawler.crawlerLogger.Println("Max depth reached:"+strconv.Itoa(depth))
		return
	}
	page:=crawler.fetchClient.FetchPage(ctx,url,crawler)
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
		    	go	crawler.crawleInside(depth,ctx,v.Resourse,resourseChans,&wg,domain)
			}else{
				crawler.crawleInside(depth,ctx,v.Resourse,resourseChans,&wg,domain)
			}
		}else{
			crawler.crawlerLogger.Println("["+resourceNode.Resourse+"]"+"Found already visited link "+v.Resourse)
		}
	}
	go func() {
			wg.Wait() 
			close(resourseChans) 
			crawler.crawlerLogger.Println("Closing chans for "+url)
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
	reHref := regexp.MustCompile(`href=\"[a-zA-Z0-9\./]+\"`)
	hrefs:=reHref.FindAll([]byte(*page),-1)
	crawler.crawlerLogger.Println(2,"Found links on page "+resorce.Resourse+":"+strconv.Itoa(len(hrefs)))
	resorce.Links=make([]ResourseNode,len(hrefs))
	for i,href:= range hrefs{
		crawler.crawlerLogger.Println(2,resorce.Resourse+":"+string(href))
		hrefParsed:=href[6 : len(href)-1]
		if strings.Contains(string(hrefParsed),*domainUrl){
			resorce.Links[i].Resourse= string(hrefParsed)
		}else{
			crawler.crawlerLogger.Println("Href of not parent domain:"+string(href)+" Parent domain:"+*domainUrl)
		}
		
	}
	return resorce
}

func extractDomain(url string) string{
	slashCounter,i:=2,0
	for ;i<len(url);i++{
		if url[i]=='\\'{
			slashCounter--;
		}
		if slashCounter<0{
			break
		}
	}
	return url[:i]
}