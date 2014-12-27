package main

import (
	"github.com/jlaffaye/ftp"
	"path"
	"path/filepath"
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
	var list []*ftp.Entry
	func() {
		mt.Lock()
		defer mt.Unlock()
		list, _ = elem.Conn.List(dir)
	}()

	for _, file := range list {
		ff := path.Join(dir, file.Name)

		// go deeper!
		if file.Type == ftp.EntryTypeFolder {
			elem.crawlDirectory(ff, mt)
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
