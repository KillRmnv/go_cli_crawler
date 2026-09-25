package clicrawler

type CliParams struct{
	params map[string] func(string)
}

func (paramsStruct* CliParams) Init(config *CrawlerConfig) {
	paramsStruct.params=map[string]func(string){
    "urls": config.SetUrls,
    "depth": config.SetDepth,
    "timeout": config.SetTimeout,
    "request-timeout":config.SetRequestTimeout,
    "output":config.SetOutput,
    "log":config.SetLog,
	}
}

func (paramsStruct* CliParams) Parse(paramsp[] string){
	
}