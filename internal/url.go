package clicrawler


type Url struct{
	adress string
}
func NewUrl(adress string) Url{
	return Url{adress: adress}
}
func(url* Url) ExtractDomain() (string,error){
	slashCounter,i:=2,0
	for ;i<len(url.adress);i++{
		if url.adress[i]=='/'{
			slashCounter--;
		}
		if slashCounter==0{
			break
		}
	}
	slashCounter=i+1
	i++
	for ;i<len(url.adress);i++{
		if url.adress[i]=='/' {
			return url.adress[slashCounter:i],nil	
		}
	}
	return  url.adress[slashCounter:],nil
}