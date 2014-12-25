package main

import (
	"flag"
	"net/http"
	"fmt"
	"log"
	"encoding/json"
	elastigo "github.com/mattbaird/elastigo/lib"
)

var (
	http_listen = flag.String("l", ":8080", "[host]:[port] where the HTTP is listening")
	es_server = flag.String("es", "localhost", "ElasticSearch host")
	es_conn *elastigo.Conn
)

func initElastics(host string) {
	es_conn = elastigo.NewConn()
	es_conn.Domain = host
}

func handler(w http.ResponseWriter, r *http.Request) {
	q := r.FormValue("q")

	searchJson := fmt.Sprintf(`{
		"query": {
			"fuzzy_like_this": {
				"fields": ["Path"],
				"like_text": "%s",
				"fuzziness": 1
			}
		}
	}`, q)
	results, err := es_conn.Search("torture", "file", nil, searchJson)
	if err != nil {
		log.Print(err)
	}

	output, err := json.Marshal(results.Hits)
	w.Write(output)
}

func main() {
	initElastics(*es_server)

	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/s", handler)
	http.ListenAndServe(*http_listen, nil)
}
