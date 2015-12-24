package main

import (
	"flag"
	"os"
)

var (
	httpListen    = flag.String("l", ":8080", "[host]:[port] where HTTP is listening")
	elasticServer = flag.String("es", "http://localhost:9200", "ElasticSearch host")
	perPage       = flag.Int("pp", 15, "Results per page")

	tlsListen = flag.String("tl", ":8043", "[host]:[port] where HTTPS is listening")
	tlsCert   = flag.String("cert", "", "Path to a TLS certificate")
	tlsKey    = flag.String("key", "", "Path to a TLS key")
)

func main() {
	flag.Parse()

	frontend, err := CreateFrontend(FrontendConfig{
		HttpListen:    *httpListen,
		ElasticServer: *elasticServer,
		PerPage:       *perPage,
		LogOutput:     os.Stdout,

		TLSListen: *tlsListen,
		TLSCert:   *tlsCert,
		TLSKey:    *tlsKey,
	})
	if err != nil {
		frontend.Log.Fatal(err)
	}
}
