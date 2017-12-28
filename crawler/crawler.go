package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"
)

// Data structures shared by all protocol-specific crawlers

type FileInfo struct {
	URL      *url.URL
	Size     int64
	MimeType string
	ModTime  time.Time
}

type WalkFunction func(path string, info FileInfo)

type Crawler interface {
	// Start walking recursively and call fn on every file
	Walk(fn WalkFunction) error

	// Tear down all open connections
	Close()
}

// Data structures only used to control instances of protocol specific crawlers

type CrawlerConfig struct {
	Entry     string        `json:"entry"`
	TurnDelay time.Duration `json:"turnDelay"`
}

type CrawlersConfig struct {
	Entrypoints []*json.RawMessage `json:"entrypoints"`
}

type CrawlerEntry struct {
	Config    CrawlerConfig
	RawConfig *json.RawMessage
	Crawler   *Crawler
	Terminate chan bool
}

type Crawlers struct {
	Config    CrawlersConfig
	Crawlers  []*CrawlerEntry
	WaitGroup sync.WaitGroup
	Model     *Model
}

func CreateCrawlers(configPath string, model *Model) (crawlers *Crawlers, err error) {
	crawlers = &Crawlers{
		Model: model,
	}

	// Initially load config
	crawlers.Load(configPath)

	// Initialize config reloading using syscall
	reloadChan := make(chan os.Signal)
	signal.Notify(reloadChan, syscall.SIGUSR1)
	go func() {
		for {
			<-reloadChan

			log.Println("reload!")
			crawlers.Load(configPath)
		}
	}()

	return
}

// (Re-)load crawler configs and create crawler instances. You can call this
// function again multiple times to reload the configs
func (crawlers *Crawlers) Load(configPath string) (err error) {
	rawConfig, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return
	}

	nextConfig := CrawlersConfig{}
	err = json.Unmarshal(rawConfig, &nextConfig)
	if err != nil {
		return
	}

	var nextCrawlers []*CrawlerEntry
	for _, entrypoint := range nextConfig.Entrypoints {
		// Parse config while providing default values
		entryConfig := CrawlerConfig{
			TurnDelay: 10 * time.Second,
		}
		err = json.Unmarshal(*entrypoint, &entryConfig)
		if err != nil {
			return
		}

		nextCrawlers = append(nextCrawlers, &CrawlerEntry{
			Config:    entryConfig,
			RawConfig: entrypoint,
		})
	}

	// 1. check for removed crawlers and terminate them
OUTER:
	for _, oldCrawler := range crawlers.Crawlers {
		for _, newCrawler := range nextCrawlers {
			if bytes.Compare([]byte(*oldCrawler.RawConfig), []byte(*newCrawler.RawConfig)) == 0 {
				// if crawler config has not changed, keep the crawler instance and continue
				newCrawler.Crawler = oldCrawler.Crawler
				continue OUTER
			}
		}

		// otherwise terminate the old crawler
		log.Printf("server %s terminated\n", oldCrawler.Config.Entry)
		oldCrawler.Terminate <- true
	}

	// 2. set new/updated crawlers
	crawlers.Config = nextConfig
	crawlers.Crawlers = nextCrawlers

	// 3. (re-)start crawlers
	for _, entry := range crawlers.Crawlers {

		crawlers.WaitGroup.Add(1)
		go func(entry *CrawlerEntry) {
			defer crawlers.WaitGroup.Done()

			// goroutine-internal error variable so we do not get in conflict with
			// the outer err variable
			var gErr error

			// Parse the entry URL so we can determine the protocol
			var entryUrl *url.URL
			entryUrl, err = url.Parse(entry.Config.Entry)
			if err != nil {
				return
			}

			// Create a protocol-specific Crawler instance
			var crawler Crawler

			switch entryUrl.Scheme {
			case "http", "https":
				crawler, gErr = CreateHttpCrawler(entry.RawConfig)
			case "ftp":
				crawler, gErr = CreateFtpCrawler(entry.RawConfig)
			default:
				gErr = fmt.Errorf("Unkonwn protocol: %s", entryUrl.Scheme)
			}

			if gErr != nil {
				log.Println(gErr)
				return
			}

			defer crawler.Close()

			// Create an infinite crawling loop
			for {
				select {
				case <-entry.Terminate:
					return
				default:
					gErr = crawler.Walk(crawlers.WalkFn)
					if gErr != nil {
						log.Println(gErr)
						// Do not terminate as of Walk errors, just keep trying
						// return
					}
					time.Sleep(entry.Config.TurnDelay)
				}
			}
		}(entry)

	}

	return
}

func (crawlers *Crawlers) WalkFn(currentPath string, info FileInfo) {
	// get url without path
	urlCpy := *info.URL
	infoPath := urlCpy.Path
	urlCpy.Path = ""
	infoUrl := urlCpy.String()

	file := ModelFileEntry{
		Filename: path.Base(info.URL.Path),
		Size:     info.Size,
		MimeType: info.MimeType,
		ModTime:  info.ModTime,
		Servers: []ModelFileServerEntry{{
			Url:  infoUrl,
			Path: infoPath,
		}},
	}

	err := crawlers.Model.AddFileEntry(file)
	if err != nil {
		log.Println(err)
	}
}

func (crawlers *Crawlers) Run() {
	crawlers.WaitGroup.Wait()
}

func (crawlers *Crawlers) Quit() {
	for _, crawler := range crawlers.Crawlers {
		crawler.Terminate <- true
	}
}
