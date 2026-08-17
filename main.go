package main

import (
	"fmt"

	"github.com/hhhug000/harvester/crawler"
)

func main() {
	var domain string
	fmt.Println("Choose a domain to scrape")
	fmt.Scan(&domain)
	crawlerEngine := crawler.NewEngine("./output")
	crawlerEngine.Crawl(domain)
}
