package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dustin/go-humanize"
	elastigo "github.com/mattbaird/elastigo/lib"
	"gopkg.in/flosch/pongo2.v3"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	http_listen = flag.String("l", ":8080", "[host]:[port] where the HTTP is listening")
	es_server   = flag.String("es", "localhost", "ElasticSearch host")
	es_conn     *elastigo.Conn
	tmpls       = make(map[string]*pongo2.Template)
)

type Result struct {
	Server    string
	Path      string
	Filename  string
	Size      uint64
	HumanSize string
}

func initElastics(host string) {
	es_conn = elastigo.NewConn()
	es_conn.Domain = host
}

func handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	q := r.FormValue("q")

	searchJson := fmt.Sprintf(`{
		"query": {
			"match": {
				"Path": {
					"query": "%s",
					"fuzziness": 1
				}
			}
		}
	}`, q)
	query_response, err := es_conn.Search("torture", "file", nil, searchJson)
	if err != nil {
		log.Print(err)
	}

	// API like a bauss
	if r.FormValue("f") == "json" {
		output, _ := json.Marshal(query_response.Hits)
		w.Header().Set("Content-Type", "application/json")
		w.Write(output)
		return
	}

	var results []Result
	for _, qr := range query_response.Hits.Hits {
		var result Result
		raw, _ := qr.Source.MarshalJSON()
		json.Unmarshal(raw, &result)

		// get the plain filename
		path_splitted := strings.Split(result.Path, "/")
		result.Filename = path_splitted[len(path_splitted)-1]
		result.HumanSize = humanize.Bytes(result.Size)

		results = append(results, result)
	}

	tmpls["results"].ExecuteWriter(pongo2.Context{
		"query":    q,
		"elapsed":  time.Since(start) / time.Millisecond,
		"response": query_response,
		"results":  results,
	}, w)
}

func main() {
	initElastics(*es_server)

	// load templates
	tmpl := pongo2.NewSet("torture")
	tmpl.SetBaseDirectory("templates")

	if tmpl, err := tmpl.FromFile("results.tmpl"); err == nil {
		tmpls["results"] = tmpl
	} else {
		log.Fatal(err)
	}

	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/s", handler)
	http.ListenAndServe(*http_listen, nil)
}
