package main

import (
	"flag"
	"log"
)

const (
	DEFAULT_BOTNAME = "TortureBot"
)

var (
	configFile    = flag.String("c", "config.json", "Config file")
	elasticServer = flag.String("es", "http://localhost:9200", "ElasticSearch host")
)

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	model, err := CreateModel(*elasticServer)
	if err != nil {
		panic(err)
	}

	crawlers, err := CreateCrawlers(*configFile, model)
	if err != nil {
		panic(err)
	}
	crawlers.Run()

	defer crawlers.Quit()
}
