package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Config struct {
	ServersFile   string
	ElasticServer string
	LogOutput     io.Writer
}

type Crawler struct {
	cfg           Config
	servers       []*Ftp
	elasticSearch *ElasticSearch
	Log           *log.Logger
}

func CreateCrawler(cfg Config) (cr *Crawler, err error) {
	cr = &Crawler{cfg: cfg}

	// Create logger
	cr.Log = log.New(cr.cfg.LogOutput, "crawler: ", log.Ldate|log.Lshortfile)

	// Create an ElasticSearch connection
	cr.elasticSearch, err = CreateElasticSearch(cr.cfg.ElasticServer, cr)
	if err != nil {
		return
	}

	// Create index and mapping
	err = cr.elasticSearch.CreateMappingAndIndex()
	if err != nil {
		return
	}

	// Start listening for the USR1 syscall to reload the FTP servers
	cr.InitReloadingFtps(syscall.SIGUSR1)

	// Load FTPs and start recursive crawling
	err = cr.LoadFtpsAndStartCrawling()
	if err != nil {
		return
	}

	return
}

// If no servers are loaded just let them load from a file.
// If some servers are already present, keep existing, remove disappeared and add new.
func (cr *Crawler) LoadFtps(fileName string) (err error) {
	servers, err := cr.ScanFtpsFromFile(fileName)
	if err != nil {
		return
	}

	// Reloading servers; Keep existing, remove disappeared, add new
	if len(cr.servers) > 0 {
		for _, oldServer := range cr.servers {
			var isIncluded bool

			// Use the existing server object for already existing servers,
			// thus re-use the existing FTP connection etc. pp.
			for i, newServer := range servers {
				if oldServer.Url == newServer.Url {
					servers[i] = oldServer
					isIncluded = true
				}
			}

			// Remove servers that disappeared from the list
			// TODO Use channels to achieve this
			if !isIncluded {
				oldServer.Obsolete = true
			}
		}
	}

	cr.servers = servers
	return
}

// Load a file, read each line and create a slice of Ftp types
func (cr *Crawler) ScanFtpsFromFile(fileName string) (ftpServers []*Ftp, err error) {
	file, err := os.Open(fileName)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ftp, err := CreateFtp(scanner.Text(), cr)
		if err != nil {
			return nil, err
		}
		ftpServers = append(ftpServers, ftp)
	}
	return
}

func (cr *Crawler) LoadFtpsAndStartCrawling() (err error) {
	// Initially load FTP servers
	err = cr.LoadFtps(cr.cfg.ServersFile)
	if err != nil {
		return
	}

	// Start crawling
	cr.StartCrawling()
	return
}

// Reload the servers list on signal sig
func (cr *Crawler) InitReloadingFtps(sig os.Signal) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, sig)

	go func(cr *Crawler) {
		<-c

		err := cr.LoadFtpsAndStartCrawling()
		if err != nil {
			cr.Log.Print(err)
		}
	}(cr)
}

// Start a Goroutine for each FTP server and wait until all are done (never happening)
func (cr *Crawler) StartCrawling() {
	var wg sync.WaitGroup

	for _, elem := range cr.servers {
		// Do not start more than one crawling goroutine per server
		if elem.Running {
			continue
		}
		elem.Running = true
		wg.Add(1)

		go func(el *Ftp) {
			defer el.Conn.Quit()

			el.ConnectLoop()
			el.LoginLoop()

			go el.NoOpLoop()

			for !el.Obsolete {
				el.StartCrawling()
			}
		}(elem)
	}

	wg.Wait()
}
