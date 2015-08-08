package main

import (
	"github.com/jlaffaye/ftp"
	"net/url"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type Ftp struct {
	URL      *url.URL
	Running  bool
	Obsolete bool
	Conn     *ftp.ServerConn

	crawler *Crawler
	mt      *sync.Mutex
}

func CreateFtp(surl string, crawler *Crawler) (ftp *Ftp, err error) {
	parsedURL, err := url.Parse(surl)
	if err != nil {
		return
	}

	ftp = &Ftp{
		URL:     parsedURL,
		crawler: crawler,
		mt:      &sync.Mutex{},
	}

	crawler.Log.Print("Added ", parsedURL.String())
	return
}

// Try to connect as long as the server is not obsolete
// This function does not return errors as high-load FTPs
// do likely need hundreds of connection retries
func (elem *Ftp) ConnectLoop() {
	for !elem.Obsolete {
		conn, err := ftp.Connect(elem.URL.Host)
		if err == nil {
			elem.Conn = conn
			break
		}

		elem.crawler.Log.Print(err)
		time.Sleep(2 * time.Second)
	}
}

// LoginLoop consciously tries to login on the ftp server.
// If the given URL specified password and/or user then those
// values will be used otherwise it will fallback to anonymous:anonymous.
// This function does not return errors as high-load FTPs
// do likely need hundreds of login retries
func (elem *Ftp) LoginLoop() {
	userInfo := elem.URL.User
	name := "anonymous"
	if len(userInfo.Username()) > 0 {
		name = userInfo.Username()
	}

	userPass, ps := userInfo.Password()
	pass := "anonymous"
	if ps {
		pass = userPass
	}

	for !elem.Obsolete {
		err := elem.Conn.Login(name, pass)
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
				Url:  elem.URL.String(),
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
