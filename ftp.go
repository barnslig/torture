package main

import (
	"github.com/jlaffaye/ftp"
	"log"
	"path"
	"sync"
)

type FTP struct {
	Url      string
	Running  bool
	Obsolete bool
	Conn     *ftp.ServerConn
}

func (elem *FTP) crawlDirectory(dir string, mt *sync.Mutex) {
	if elem.Obsolete {
		return
	}
	var (
		list []*ftp.Entry
		err  error
	)
	func() {
		mt.Lock()
		defer mt.Unlock()
		list, err = elem.Conn.List(dir)
	}()
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range list {
		ff := path.Join(dir, file.Name)

		// go deeper!
		if file.Type == ftp.EntryTypeFolder {
			elem.crawlDirectory(ff, mt)
		}
		// into teh elastics
		if file.Type == ftp.EntryTypeFile {
			fe := FileEntry{
				elem.Url,
				ff,
				file.Size,
			}
			addToElastic(fe)
		}
	}
}

func (elem *FTP) crawlFtpDirectories(mt *sync.Mutex) {
	pwd, _ := elem.Conn.CurrentDir()
	for !elem.Obsolete {
		elem.crawlDirectory(pwd, mt)
	}
}
