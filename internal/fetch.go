package clicrawler

import (
	"bufio"
	"context"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)
type FetchClient struct{
	client http.Client // потокобезопасен
	errStatusLogger log.Logger // базовая реализация логгера потокобезопасна
	semaphore       chan struct{}
}
const maxBodySize = 5 * 1024 * 1024

var skipExt = map[string]struct{}{
	".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".webp": {}, ".svg": {}, ".ico": {},
	".pdf": {}, ".zip": {}, ".rar": {}, ".7z": {}, ".gz": {}, ".tar": {},
	".mp3": {}, ".mp4": {}, ".avi": {}, ".mkv": {}, ".mov": {},
	".exe": {}, ".msi": {}, ".dmg": {}, ".iso": {}, ".apk": {},
	".doc": {}, ".docx": {}, ".xls": {}, ".xlsx": {}, ".ppt": {}, ".pptx": {},
	".css": {}, ".js": {}, ".woff": {}, ".woff2": {}, ".ttf": {},
}

func hasSkippedExt(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	_, skip := skipExt[strings.ToLower(path.Ext(u.Path))]
	return skip
}

func mediaType(ct string) string {
	mt, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return ""
	}
	return mt
}

func isHTMLType(mt string) bool {
	return mt == "text/html" || mt == "application/xhtml+xml"
}

func needsSniff(mt string) bool {
	return mt == "" || mt == "text/plain" || mt == "application/octet-stream"
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

type FetchPolicy interface{
	FetchLogger() *log.Logger
	MaxRetry() int
	RetryDelay() time.Duration
	MarkVisited(url string)
}
//планировщик go самостоятельно распределит ресурсы между корутинами, поэтому тут не вижу смысла параллелить
func(client* FetchClient) FetchPage(ctx context.Context,url string,policy FetchPolicy) string{
	if ctx.Err()!=nil{
		return ""
	}
	policy.FetchLogger().Println("Trying to fetch:"+url)
	if hasSkippedExt(url) {
		policy.FetchLogger().Println("Skip by extension:" + url)
		return ""
	}
	select {
		case client.semaphore <- struct{}{}: 
			defer func() { <-client.semaphore }()
			policy.FetchLogger().Println("Fetch started:" + url)
		case <-ctx.Done():
			policy.FetchLogger().Println("Gracefully stopping fetching (in queue)")
			return ""
	}
	
	retryAmount := 0
	for {
		select {
		case <-ctx.Done():
			policy.FetchLogger().Println("Gracefully stopping fetching")
			return ""
		default:
			if retryAmount <= policy.MaxRetry() {
				result, isContinue := client.processGet(ctx, url, policy, &retryAmount)
				if isContinue {
					continue 
				}
				return result 
			} else {
				policy.FetchLogger().Println("Can not reach resource after retries:" + url)
				policy.MarkVisited(url)
				return ""
			}
		}
	}	
}

func(client* FetchClient) processGet(ctx context.Context, url string, policy FetchPolicy, retryAmount* int) (string, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err!=nil{
		client.errStatusLogger.Printf("Can not build request for:%s (%s)", url, err.Error())
		return "", false
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,*/*;q=0.8")

	resp, err := client.client.Do(req)
	if err != nil {
		client.errStatusLogger.Printf("Error while GET request:%s", err.Error())
		*retryAmount++
		if *retryAmount > policy.MaxRetry(){
			return "", true
		}
		select {
		case <-time.After(policy.RetryDelay()):
			policy.FetchLogger().Println("Gorutine try's again after delay:" + url)
			return "", true 
		case <-ctx.Done():
			policy.FetchLogger().Println("Gracefully stopping waiting")
			return "", false
		}
	}
	defer resp.Body.Close()
	
	policy.MarkVisited(url)
	client.errStatusLogger.Printf("Request status code:%s; Url:%s", resp.Status, url)
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		policy.FetchLogger().Println("Skip redirect:" + url)
		return "", false
	}
	if resp.StatusCode >= 400 {
		*retryAmount++
		return "", true
	}

	mt := mediaType(resp.Header.Get("Content-Type"))
	if !isHTMLType(mt) && !needsSniff(mt) {
		policy.FetchLogger().Println("Skip non HTML resource:" + url + " Type:" + mt)
		return "", false
	}

	if resp.ContentLength > maxBodySize {
		policy.FetchLogger().Println("Skip too large resource:" + url)
		return "", false
	}
	
	br := bufio.NewReaderSize(resp.Body, 1024)
	if !isHTMLType(mt) {
		head, _ := br.Peek(512)
		if !strings.HasPrefix(http.DetectContentType(head), "text/html") {
			policy.FetchLogger().Println("Skip by sniffing:" + url)
			return "", false
		}
	}

	body, err := io.ReadAll(io.LimitReader(br, maxBodySize+1))
	if err != nil {
		policy.FetchLogger().Println("Error while reading body:" + err.Error())
		return "", false
	}
	if len(body) > maxBodySize {
		policy.FetchLogger().Println("Body exceeds limit:" + url)
		return "", false
	}
	return string(body), false
}
