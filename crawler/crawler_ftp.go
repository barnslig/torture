package main

import (
	"encoding/json"
	"github.com/jlaffaye/ftp"
	"github.com/temoto/robotstxt"
	"io/ioutil"
	"mime"
	"net/url"
	"path"
	"sync"
	"time"
)

type FtpCrawlerConfig struct {
	Entry         string        `json:"entry"`
	RateLimit     time.Duration `json:"maxRequestPerSecond"`
	RobotName     string        `json:"robotName"`
	ObeyRobotsTxt bool          `json:"obeyRobotsTxt"`
}

type FtpCrawler struct {
	Config FtpCrawlerConfig

	AuthPass string
	AuthUser string
	Entry    *url.URL

	Conn            *ftp.ServerConn
	ConnMt          sync.Mutex
	RobotsTestAgent *robotstxt.Group
	Terminate       chan bool
	Ticker          <-chan time.Time
}

func CreateFtpCrawler(rawConfig *json.RawMessage) (crawler *FtpCrawler, err error) {
	// Create a new instance
	crawler = &FtpCrawler{
		AuthPass: "anonymous",
		AuthUser: "anonymous",

		Terminate: make(chan bool, 1),
	}

	// Parse config while providing default values
	config := FtpCrawlerConfig{
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

	// Try to parse username and password from the URL
	userInfo := crawler.Entry.User
	if userInfo != nil {
		crawler.AuthUser = userInfo.Username()
		if _pass, ok := userInfo.Password(); ok {
			crawler.AuthPass = _pass
		}
	}

	// Determine port
	port := "21"
	if crawler.Entry.Port() != "" {
		port = crawler.Entry.Port()
	}

	// Try to connect in a loop. High-Load FTPs likely need a few hundred tries
	addr := crawler.Entry.Hostname() + ":" + port
	for {
		conn, err := ftp.Dial(addr)
		if err == nil {
			crawler.Conn = conn
			break
		}

		// TODO gradually slow down
		time.Sleep(2 * time.Second)
	}

	// Try to log-in in a loop. High-Load FTPs likely need a few hundred tries
	for {
		err := crawler.Conn.Login(crawler.AuthUser, crawler.AuthPass)
		if err == nil {
			break
		}

		// TODO gradually slow down
		time.Sleep(2 * time.Second)
	}

	// Give FTP some time to get ready, e.g. finish sending a motd. Otherwise it
	// might happen that the FTP library interprets motd content as command
	// response.
	time.Sleep(5 * time.Second)

	// Try to parse robots.txt
	if crawler.Config.ObeyRobotsTxt {
		robotsRes, robotsErr := crawler.Conn.Retr("/robots.txt")
		if robotsErr == nil {
			defer robotsRes.Close()

			var robotsData []byte
			robotsData, err = ioutil.ReadAll(robotsRes)
			if err != nil {
				return
			}

			var robots *robotstxt.RobotsData
			robots, err = robotstxt.FromBytes(robotsData)
			if err != nil {
				return
			}

			crawler.RobotsTestAgent = robots.FindGroup(crawler.Config.RobotName)
		}
	}

	// Repeatedly send NoOps to prevent the connection from closing
	go func() {
		for {
			select {
			case <-crawler.Terminate:
				return
			default:
				crawler.ConnMt.Lock()
				crawler.Conn.NoOp()
				crawler.ConnMt.Unlock()
				time.Sleep(15 * time.Second)
			}
		}
	}()

	return
}

func (crawler *FtpCrawler) walker(entry *url.URL, fn WalkFunction) (err error) {
	// Check if this file is allowed to be crawled by robots.txt rules
	if crawler.RobotsTestAgent != nil && !crawler.RobotsTestAgent.Test(entry.String()) {
		return
	}

	// Throttle requests as specified by the RateLimit if needed
	if crawler.Config.RateLimit > 0 {
		<-crawler.Ticker
	}

	crawler.ConnMt.Lock()
	files, err := crawler.Conn.List(entry.Path)
	crawler.ConnMt.Unlock()
	if err != nil {
		return
	}

	for _, file := range files {
		// only go deeper
		if file.Name == "." || file.Name == ".." {
			continue
		}

		entryUrl := *entry
		entryUrl.Path = path.Join(entry.Path, file.Name)

		// We only continue walking on directories
		if file.Type == ftp.EntryTypeFile {
			// TODO find out the mime type using magic numbers
			fn("", FileInfo{
				URL:      &entryUrl,
				Size:     int64(file.Size),
				MimeType: mime.TypeByExtension(path.Ext(entryUrl.Path)),
				ModTime:  file.Time,
			})
			continue
		}

		err = crawler.walker(&entryUrl, fn)
		if err != nil {
			return
		}
	}

	return
}

func (crawler *FtpCrawler) Walk(fn WalkFunction) error {
	return crawler.walker(crawler.Entry, fn)
}

func (crawler *FtpCrawler) Close() {
	crawler.Terminate <- true
	crawler.Conn.Quit()
}
