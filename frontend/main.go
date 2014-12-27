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
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	http_listen = flag.String("l", ":8080", "[host]:[port] where the HTTP is listening")
	es_server   = flag.String("es", "localhost", "ElasticSearch host")
	per_page    = flag.Int("pp", 15, "Results per page")
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

type hash map[string]interface{}

func initElastics(host string) {
	es_conn = elastigo.NewConn()
	es_conn.Domain = host
}

func getPageLink(page int, url *url.URL) string {
	if page < 1 {
		page = 0
	}

	purl := url
	qry := purl.Query()
	qry.Set("p", strconv.Itoa(page))
	purl.RawQuery = qry.Encode()
	return purl.String()
}

func handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	q := r.FormValue("q")
	p, _ := strconv.Atoi(r.FormValue("p"))

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
	query_response, err := es_conn.Search("torture", "file", hash{
		"size": *per_page,
		"from": *per_page * p,
	}, searchJson)
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
		"query": q,

		"page":     p,
		"frompage": *per_page * p,
		"maxpages": query_response.Hits.Total / *per_page,
		"prevpage": getPageLink(p-1, r.URL),
		"nextpage": getPageLink(p+1, r.URL),

		"elapsed":  time.Since(start) / time.Millisecond,
		"response": query_response,
		"results":  results,
	}, w)
}

func main() {
	flag.Parse()

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
