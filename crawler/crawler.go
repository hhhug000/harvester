package crawler

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

type Engine struct {
	outputDir   string
	scrapeCount int
}

func NewEngine(outputDir string) *Engine {
	return &Engine{outputDir: outputDir, scrapeCount: 0}
}

func (e *Engine) logPage(el *colly.HTMLElement) {
	e.scrapeCount += 1
	host := el.Request.URL.Host
	path := el.Request.URL.Path
	log.Printf("[\033[35m%d\033[0m] \033[32mScraped:\033[0m \033[36m%s\033[0m\033[33m%s\033[0m", e.scrapeCount, host, path)
}

func (e *Engine) convertHTMLtoMD(htmlStr string) (string, error) {
	converter := md.NewConverter("", true, nil)

	markdown, err := converter.ConvertString(htmlStr)
	if err != nil {
		return "", err
	}

	return markdown, nil
}

func (e *Engine) processPage(el *colly.HTMLElement) {
	domain := el.Request.URL.Host
	urlPath := el.Request.URL.Path

	el.DOM.Find("script, style, nav, footer, header").Remove()

	mainContent := el.DOM.Find("main, article, body").First()

	currentURL := el.Request.URL
	currDir := filepath.FromSlash(currentURL.Path)

	mainContent.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "mailto:") {
			return
		}

		targetURL, err := currentURL.Parse(href)
		if err != nil || targetURL.Host != currentURL.Host {
			return
		}

		targetFile := filepath.Join(filepath.FromSlash(targetURL.Path), "index.md")

		relPath, err := filepath.Rel(currDir, targetFile)
		if err != nil {
			return
		}

		relURL := filepath.ToSlash(relPath)

		if targetURL.RawQuery != "" {
			relURL += "?" + targetURL.RawQuery
		}
		if targetURL.Fragment != "" {
			relURL += "#" + targetURL.Fragment
		}

		s.SetAttr("href", relURL)
	})

	htmlStr, err := mainContent.Html()
	if err != nil {
		return
	}

	finalFilePath := filepath.Join(e.outputDir, domain, filepath.FromSlash(urlPath), "index.md")

	absBase, _ := filepath.Abs(filepath.Join(e.outputDir, domain))
	absFinal, _ := filepath.Abs(finalFilePath)
	if !strings.HasPrefix(absFinal, absBase) {
		return
	}

	dirPath := filepath.Dir(finalFilePath)
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		return
	}

	markdown, err := e.convertHTMLtoMD(htmlStr)
	if err != nil {
		return
	}

	_ = os.WriteFile(finalFilePath, []byte(markdown), 0644)
}

func (e *Engine) Crawl(domain string) {
	c := colly.NewCollector(
		colly.AllowedDomains(domain),
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*" + domain + "*",
		Parallelism: 10,
	})

	c.OnHTML("a[href]", func(el *colly.HTMLElement) {
		link := el.Attr("href")
		el.Request.Visit(link)
	})

	c.OnHTML("html", func(el *colly.HTMLElement) {
		e.logPage(el)
		e.processPage(el)
	})

	c.Visit("https://" + domain)

	c.Wait()
}
