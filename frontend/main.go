package main

import (
	"flag"
	"os"
)

var (
	httpListen    = flag.String("l", ":8080", "[host]:[port] where the HTTP is listening")
	elasticServer = flag.String("es", "localhost", "ElasticSearch host")
	perPage       = flag.Int("pp", 15, "Results per page")
)

func main() {
	flag.Parse()

	frontend, err := CreateFrontend(FrontendConfig{
		HttpListen:    *httpListen,
		ElasticServer: *elasticServer,
		PerPage:       *perPage,
		LogOutput:     os.Stdout,
	})
	if err != nil {
		frontend.Log.Fatal(err)
	}
}
