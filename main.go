package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/jlaffaye/ftp"
	"os"
	"sync"
	"time"
)

var (
	servers_file = flag.String("f", "servers.txt", "file with one ftp per line")
	es_server    = flag.String("es", "localhost", "ElasticSearch host")
	servers      []FTP
)

func loadFTPs() {
	if len(servers) <= 0 {
		servers, _ = scanServers()
		return
	}

	newServers, _ := scanServers()
	for _, oldServer := range servers {
		var isIncluded bool
		for _, newServer := range newServers {
			if oldServer.Url == newServer.Url {
				isIncluded = true
			}
		}

		if !isIncluded {
			oldServer.Obsolete = true
		}
	}
}

func scanServers() (servers []FTP, err error) {
	var ftpServers []FTP

	file, err := os.Open(*servers_file)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ftp := FTP{
			scanner.Text(),
			false,
			false,
			nil,
		}

		servers = append(ftpServers, ftp)
	}

	return
}

func startFTPConnCycler() {
	var wg sync.WaitGroup

	for _, elem := range servers {
		if elem.Running {
			break
		}

		wg.Add(1)
		go func(el FTP) {
			fmt.Println(el.Url)
			var mt = &sync.Mutex{}

			// try to connect
			for {
				conn, err := ftp.Connect(el.Url)
				if err == nil {
					el.Conn = conn
					fmt.Println("connected!")
					break
				}
				fmt.Println("retry …")
				time.Sleep(2 * time.Second)
			}

			// try to log in as anonymous
			if el.Conn.Login("anonymous", "anonymous") != nil {
				fmt.Println("Login as anonymous failed.")
				wg.Done()
			}
			el.Running = true

			// start a goroutine that sends a NoOp every 15 seconds
			go func(el FTP) {
				for !el.Obsolete {
					time.Sleep(15 * time.Second)
					fmt.Println("noop")

					func() {
						mt.Lock()
						defer mt.Unlock()
						el.Conn.NoOp()
					}()
				}
			}(el)

			el.crawlFtpDirectories(mt)
			el.Conn.Quit()
		}(elem)
	}

	wg.Wait()
}

func main() {
	flag.Parse()

	initElastics(*es_server)

	loadFTPs()
	startFTPConnCycler()
}
