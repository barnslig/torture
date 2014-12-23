package main

import (
	"flag"
	"os"
	"bufio"
	"log"
	"fmt"
	"sync"
	"time"
	"github.com/jlaffaye/ftp"
)

var (
	servers_file = flag.String("f", "servers.txt", "file with one ftp per line")
	servers []FTP
)

func loadFTPs() {
	file, err := os.Open(*servers_file)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ftp := FTP{
			scanner.Text(),
			false,
			nil,
		}
		servers = append(servers, ftp)
	}
}

func startFTPConnCycler() {
	var wg sync.WaitGroup

	for _, elem := range servers {
		wg.Add(1)

		go func(el FTP) {
			el.running = true
			fmt.Println(el.url)

			var mt = &sync.Mutex{}

			// try to connect
			for {
				conn, err := ftp.Connect(el.url)
				if err == nil {
					el.conn = conn
					fmt.Println("connected!")
					break
				}
				fmt.Println("retry …")
				time.Sleep(2 * time.Second)
			}

			// try to log in as anonymous
			if el.conn.Login("anonymous", "anonymous") != nil {
				fmt.Println("Login as anonymous failed.")
				el.running = false
				wg.Done()
			}

			// start a goroutine that sends a NoOp every 15 seconds
			go func(conn *ftp.ServerConn) {
				for {
					time.Sleep(15 * time.Second)
					fmt.Println("noop")

					func() {
						mt.Lock()
						defer mt.Unlock()
						conn.NoOp()
					}()
				}
			}(el.conn)

			el.crawlFtpDirectories(mt)
		}(elem)
	}

	wg.Wait()
}

func main() {
	flag.Parse()

	loadFTPs()
	startFTPConnCycler()
}
