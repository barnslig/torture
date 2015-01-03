package main

import (
	"flag"
	"os"
)

var (
	serversFile   = flag.String("f", "servers.txt", "File with one ftp per line")
	elasticServer = flag.String("es", "localhost", "ElasticSearch host")
)

func main() {
	flag.Parse()

	cr, err := CreateCrawler(Config{
		ServersFile:   *serversFile,
		ElasticServer: *elasticServer,
		LogOutput:     os.Stdout,
	})
	if err != nil {
		cr.Log.Fatal(err)
	}
}
