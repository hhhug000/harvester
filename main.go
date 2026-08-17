package main

import (
	"github.com/hhhug000/harvester/crawler"
)

func main() {
	crawlerEngine := crawler.NewEngine("./output")
	crawlerEngine.Crawl("go.dev")
}
