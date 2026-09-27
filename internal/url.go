package clicrawler

import (
	"errors"
	neturl "net/url"
)

type Url struct{
	adress string
}
func NewUrl(adress string) Url{
	return Url{adress: adress}
}
func(url* Url) ExtractDomain() (string,error){
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
	return parsed.Host,nil
}