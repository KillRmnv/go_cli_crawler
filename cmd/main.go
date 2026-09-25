package main

import (

	"os"
	"crawler/internal"
	
)


func main() {
	params:=clicrawler.CliParams{}
	config:=clicrawler.CrawlerConfig{}
	params.Init(&config)
	params.Parse(os.Args[1:])
	
}