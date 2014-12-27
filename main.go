package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/jlaffaye/ftp"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	servers_file = flag.String("f", "servers.txt", "file with one ftp per line")
	es_server    = flag.String("es", "localhost", "ElasticSearch host")
	servers      []FTP
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

func initReloading(sig os.Signal) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, sig)
	go func() {
		<-c

		loadFTPs()
		startFTPConnCycler()
	}()
}

func startFTPConnCycler() {
	var wg sync.WaitGroup

	for _, elem := range servers {
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
			}(el.Conn)

			el.crawlFtpDirectories(mt)
		}(elem)
	}

	wg.Wait()
}

func main() {
	flag.Parse()

	initElastics(*es_server)
	loadFTPs()
	initReloading(syscall.SIGUSR1)
	startFTPConnCycler()
}
