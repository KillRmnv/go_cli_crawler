package clicrawler

import (
	"errors"
	"net"
	neturl "net/url"
	"strings"
)

type Url struct{
	adress string
}
func NewUrl(adress string) Url{
	return Url{adress: adress}
}
func(url* Url) SiteHost() (string,error){
	if url.adress==""{
		return "",errors.New("empty url")
	}
	parsed,err:=neturl.Parse(url.adress)
	if err!=nil{
		return "",err
	}
	if parsed.Scheme==""{
		return "",errors.New("no scheme in url: "+url.adress)
	}
	if parsed.Host==""{
		return "",errors.New("no host in url: "+url.adress)
	}
	host:= normalizeHost(parsed.Host)
	if host==""{
		return "",errors.New("no host in url: "+url.adress)
	}
	return host,nil
}

func normalizeHost(host string) string{
	host= strings.ToLower(strings.TrimSpace(host))
	if withoutPort,_,err:= net.SplitHostPort(host); err==nil{
		host= withoutPort
	}
	host= strings.TrimSuffix(strings.Trim(host,"[]"),".")
	return strings.TrimSpace(host)
}

func hostInScope(host string, site string) bool{
	host, site = normalizeHost(host), normalizeHost(site)
	if host=="" || site==""{
		return false
	}
	if net.ParseIP(site)!=nil{
		return host==site 
	}
	if net.ParseIP(host)!=nil{
		return host==site
	}
	if !strings.Contains(site,"."){
		return host==site 
	}
	return host==site || strings.HasSuffix(host,"."+site)
}