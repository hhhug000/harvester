package crawler

import (
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

type Config struct {
	DisableDomainLock bool
	MaxDepth          int
}

type Option func(*Config)

func WithDisableDomainLock(disable bool) Option {
	return func(c *Config) {
		c.DisableDomainLock = disable
	}
}

func WithMaxDepth(depth int) Option {
	return func(c *Config) {
		c.MaxDepth = depth
	}
}

type Engine struct {
	outputDir   string
	scrapeCount int64
	config      Config
	converter   *md.Converter
}

func NewEngine(outputDir string, opts ...Option) *Engine {
	cfg := Config{
		DisableDomainLock: false,
		MaxDepth:          0,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return &Engine{
		outputDir:   outputDir,
		scrapeCount: 0,
		config:      cfg,
		converter:   md.NewConverter("", true, nil),
	}
}

func (e *Engine) logPage(el *colly.HTMLElement) {
	count := atomic.AddInt64(&e.scrapeCount, 1)
	host := el.Request.URL.Host
	path := el.Request.URL.Path
	log.Printf("[\033[35m%d\033[0m] \033[32mScraped:\033[0m \033[36m%s\033[0m\033[33m%s\033[0m", count, host, path)
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
		if err != nil {
			return
		}

		if !e.config.DisableDomainLock && targetURL.Host != currentURL.Host {
			return
		}

		var targetFile string
		if targetURL.Host == currentURL.Host {
			targetFile = filepath.Join(filepath.FromSlash(targetURL.Path), "index.md")
		} else {
			targetFile = filepath.Join("..", targetURL.Host, filepath.FromSlash(targetURL.Path), "index.md")
		}

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
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return
	}

	markdown, err := e.converter.ConvertString(htmlStr)
	if err != nil {
		return
	}

	_ = os.WriteFile(finalFilePath, []byte(markdown), 0644)
}

func (e *Engine) Crawl(rawURL string) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("Invalid URL: %v", err)
		return
	}

	var collectorOpts []colly.CollectorOption
	collectorOpts = append(collectorOpts, colly.Async(true))

	if !e.config.DisableDomainLock {
		collectorOpts = append(collectorOpts, colly.AllowedDomains(parsedURL.Host))
	}

	c := colly.NewCollector(collectorOpts...)

	if e.config.MaxDepth > 0 {
		c.MaxDepth = e.config.MaxDepth
	}

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 8,
	})

	c.OnHTML("a[href]", func(el *colly.HTMLElement) {
		link := el.Attr("href")
		el.Request.Visit(link)
	})

	c.OnHTML("html", func(el *colly.HTMLElement) {
		e.logPage(el)
		e.processPage(el)
	})

	c.Visit(parsedURL.String())
	c.Wait()
}
