package main

import (
	"github.com/jlaffaye/ftp"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type Ftp struct {
	Url      string
	Running  bool
	Obsolete bool
	Conn     *ftp.ServerConn

	crawler *Crawler
	mt      *sync.Mutex
}

func CreateFtp(url string, crawler *Crawler) (ftp *Ftp, err error) {
	ftp = &Ftp{
		Url:     url,
		crawler: crawler,
		mt:      &sync.Mutex{},
	}

	crawler.Log.Print("Added ", url)
	return
}

// Try to connect as long as the server is not obsolete
// This function does not return errors as high-load FTPs
// do likely need hundreds of connection retries
func (elem *Ftp) ConnectLoop() {
	for !elem.Obsolete {
		conn, err := ftp.Connect(elem.Url)
		if err == nil {
			elem.Conn = conn
			break
		}

		elem.crawler.Log.Print(err)
		time.Sleep(2 * time.Second)
	}
}

// Try to login as anonymous user
// This function does not return errors as high-load FTPs
// do likely need hundreds of login retries
func (elem *Ftp) LoginLoop() {
	for !elem.Obsolete {
		err := elem.Conn.Login("anonymous", "anonymous")
		if err == nil {
			break
		}

		elem.crawler.Log.Print(err)
		time.Sleep(2 * time.Second)
	}
}

// Send NoOps every 15 seconds so the connection is not closing
func (elem *Ftp) NoOpLoop() {
	for !elem.Obsolete {
		time.Sleep(15 * time.Second)

		func(elem *Ftp) {
			elem.mt.Lock()
			defer elem.mt.Unlock()

			elem.Conn.NoOp()
		}(elem)
	}
}

// Recursively walk through all directories
func (elem *Ftp) StartCrawling() (err error) {
	pwd, err := elem.Conn.CurrentDir()
	if err != nil {
		return
	}

	for !elem.Obsolete {
		elem.crawlDirectoryRecursive(pwd)
	}
	return
}

func (elem *Ftp) crawlDirectoryRecursive(dir string) {
	if elem.Obsolete {
		return
	}

	var list []*ftp.Entry

	func(elem *Ftp) {
		var err error

		elem.mt.Lock()
		defer elem.mt.Unlock()

		list, err = elem.Conn.List(dir)
		if err != nil {
			elem.crawler.Log.Print(err)
		}
	}(elem)

	for _, file := range list {
		ff := path.Join(dir, file.Name)

		// go deeper!
		if file.Type == ftp.EntryTypeFolder {
			elem.crawlDirectoryRecursive(ff)
		}

		// into teh elastics
		if file.Type == ftp.EntryTypeFile {
			var fservers []FtpEntry
			fservers = append(fservers, FtpEntry{
				Url:  elem.Url,
				Path: ff,
			})

			fe := FileEntry{
				Servers:  fservers,
				Filename: filepath.Base(ff),
				Size:     file.Size,
			}

			_, err := elem.crawler.elasticSearch.AddFileEntry(fe)
			if err != nil {
				elem.crawler.Log.Print(err)
			}
		}
	}
}
