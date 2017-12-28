package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/temoto/robotstxt"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

// Get the depth of a file path
func pathDepth(filePath string) int {
	return len(strings.Split(strings.Trim(path.Clean(filePath), "/"), "/"))
}

type HttpCrawlerConfig struct {
	BodySizeLimit int64         `json:"maxBodySize"`
	Entry         string        `json:"entry"`
	MaxPathDepth  int           `json:"maxPathDepth"`
	RateLimit     time.Duration `json:"maxRequestPerSecond"`
	RobotName     string        `json:"robotName"`
	ObeyRobotsTxt bool          `json:"obeyRobotsTxt"`
}

type HttpCrawler struct {
	Config HttpCrawlerConfig

	Entry *url.URL

	RobotsTestAgent *robotstxt.Group
	Ticker          <-chan time.Time
}

func CreateHttpCrawler(rawConfig *json.RawMessage) (crawler *HttpCrawler, err error) {
	// Create a new instance
	crawler = &HttpCrawler{}

	// Parse config while providing default values
	config := HttpCrawlerConfig{
		BodySizeLimit: 10 * 1000 * 1000, // 1 MB
		MaxPathDepth:  20,
		RateLimit:     0,
		RobotName:     DEFAULT_BOTNAME,
		ObeyRobotsTxt: true,
	}
	err = json.Unmarshal(*rawConfig, &config)
	if err != nil {
		return
	}

	crawler.Config = config

	// Parse the entry URL
	entry, err := url.Parse(crawler.Config.Entry)
	if err != nil {
		return
	}
	crawler.Entry = entry

	// Initialize the RateLimit throttle ticker if needed
	if crawler.Config.RateLimit > 0 {
		crawler.Ticker = time.Tick(crawler.Config.RateLimit)
	}

	// Try to parse /robots.txt
	if crawler.Config.ObeyRobotsTxt {
		robotsURL := url.URL{}
		robotsURL = *entry
		robotsURL.Path = "/robots.txt"

		robotsRes, robotsErr := http.Get(robotsURL.String())
		if robotsErr == nil {
			defer robotsRes.Body.Close()

			var robots *robotstxt.RobotsData
			robots, err = robotstxt.FromResponse(robotsRes)
			if err != nil {
				return
			}

			crawler.RobotsTestAgent = robots.FindGroup(crawler.Config.RobotName)
		}
	}

	return
}

func (crawler *HttpCrawler) walker(entry *url.URL, fn WalkFunction) (err error) {
	entryStr := entry.String()

	// Check if this file is allowed to be crawled by robots.txt rules
	if crawler.RobotsTestAgent != nil && !crawler.RobotsTestAgent.Test(entryStr) {
		return
	}

	// Throttle requests as specified by the RateLimit if needed
	if crawler.Config.RateLimit > 0 {
		<-crawler.Ticker
	}

	// Do a standard HTTP GET request, but only download the first few kilobytes
	// of body data. Reasons:
	// 1. Save a request as otherwise we would have to create one HEAD request
	//    before every GET to check for the content type and content length
	// 2. Work around broken HEAD implementations
	resp, err := http.Get(entryStr)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	// Determine the content length in bytes
	var contentLength int64
	if len(resp.Header.Get("Content-Length")) > 0 {
		contentLength, err = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 0)
		if err != nil {
			return
		}
	}

	// We only continue walking on Content-Type: text/html files and call the
	// WalkFunction on all other files
	// TODO implement fallback mime sniffing (magic numbers parsing)
	if resp.Header.Get("Content-Type") != "text/html" {
		var modTime time.Time
		lastModified := resp.Header.Get("Last-Modified")
		if lastModified != "" {
			modTime, err = http.ParseTime(resp.Header.Get("Last-Modified"))
			if err != nil {
				fmt.Println(resp.Header.Get("Last-Modified"))
				return
			}
		}

		fn(entryStr, FileInfo{
			URL:      entry,
			Size:     contentLength,
			MimeType: resp.Header.Get("Content-Type"),
			ModTime:  modTime,
		})

		return
	}

	// Limit the amount of downloaded data
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, crawler.Config.BodySizeLimit))
	if err != nil {
		return
	}

	if int64(len(body)) < contentLength {
		err = fmt.Errorf("BodySizeLimit exceeded")
		return
	}

	// Search for all anchor tags
	tokenizer := html.NewTokenizer(bytes.NewBuffer(body))

	for {
		tt := tokenizer.Next()

		switch tt {
		case html.ErrorToken:
			if tErr := tokenizer.Err(); tErr != io.EOF {
				err = tErr
			}
			return
		case html.StartTagToken:
			token := tokenizer.Token()

			if strings.ToLower(token.Data) == "a" {
				for _, a := range token.Attr {
					if a.Key == "href" {
						var u *url.URL
						u, err = url.Parse(a.Val)
						if err != nil {
							return
						}

						nextUrl := entry.ResolveReference(u)

						// Stop if we are going to leave the server
						if nextUrl.Hostname() != entry.Hostname() {
							break
						}

						// Stop if the path depth is decreasing
						// TODO what happens at . and ..
						if pathDepth(nextUrl.Path) < pathDepth(entry.Path) {
							break
						}

						// Stop if we have reached the maximum path depth
						if pathDepth(nextUrl.Path) > crawler.Config.MaxPathDepth {
							err = fmt.Errorf("MaxPathDepth exceeded")
							return
						}

						// Errors are bubbled up
						err = crawler.walker(nextUrl, fn)
						if err != nil {
							return
						}

						break
					}
				}
			}
		}
	}

	return
}

func (crawler *HttpCrawler) Walk(fn WalkFunction) error {
	return crawler.walker(crawler.Entry, fn)
}

func (crawler *HttpCrawler) Close() {

}
