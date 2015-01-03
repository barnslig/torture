package main

import (
	"encoding/json"
	"flag"
	elastigo "github.com/barnslig/elastigo/lib"
	"github.com/dustin/go-humanize"
	"gopkg.in/flosch/pongo2.v3"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var (
	http_listen = flag.String("l", ":8080", "[host]:[port] where the HTTP is listening")
	es_server   = flag.String("es", "localhost", "ElasticSearch host")
	per_page    = flag.Int("pp", 15, "Results per page")
	es_conn     *elastigo.Conn
	tmpls       = make(map[string]*pongo2.Template)
)

type Server struct {
	Url  string
	Path string
}

type Result struct {
	Servers   []Server
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

func search(query string, page int) (resp elastigo.SearchResult, err error) {
	searchQ := hash{
		"query": hash{
			"match": hash{
				"Path": hash{
					"query":     query,
					"fuzziness": 1,
				},
			},
		},
	}

	resp, err = es_conn.Search("torture", "file", hash{
		"size": *per_page,
		"from": *per_page * page,
	}, searchQ)

	return
}

func handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		if err, ok := recover().(error); ok {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()

	query := r.FormValue("q")
	page, err := strconv.Atoi(r.FormValue("p"))
	if err != nil {
		panic(err)
	}

	resp, err := search(query, page)
	if err != nil {
		panic(err)
	}

	// API like a bauss
	if r.FormValue("f") == "json" {
		output, err := json.Marshal(resp.Hits)
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(output)
		if err != nil {
			panic(err)
		}

		return
	}

	var results []Result
	for _, qr := range resp.Hits.Hits {
		raw, err := qr.Source.MarshalJSON()
		if err != nil {
			panic(err)
		}

		var result Result
		err = json.Unmarshal(raw, &result)
		if err != nil {
			panic(err)
		}

		result.HumanSize = humanize.Bytes(result.Size)
		results = append(results, result)
	}

	tmpls["results"].ExecuteWriter(pongo2.Context{
		"query":    query,

		"page":     page,
		"frompage": *per_page * page,
		"maxpages": resp.Hits.Total / *per_page,
		"prevpage": getPageLink(page-1, r.URL),
		"nextpage": getPageLink(page+1, r.URL),

		"elapsed":  time.Since(start) / time.Millisecond,
		"response": resp,
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

	http.HandleFunc("/s", handler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/", http.RedirectHandler("/s", 301))
	log.Fatal(http.ListenAndServe(*http_listen, nil))
}
