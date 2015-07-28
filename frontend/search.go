package main

import (
	"encoding/json"
	"github.com/dustin/go-humanize"
	"github.com/flosch/pongo2"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
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

type SearchConfig struct {
	Frontend *Frontend
}

type Search struct {
	cfg  SearchConfig
	tmpl *pongo2.Template
}

func CreateSearch(cfg SearchConfig) (search *Search, err error) {
	search = &Search{cfg: cfg}

	// Load the results template
	search.tmpl, err = search.cfg.Frontend.templates.FromFile("results.tmpl")
	if err != nil {
		return
	}

	return
}

func (search *Search) Handler(w http.ResponseWriter, r *http.Request) {
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
		page = 0
	}

	resp, err := search.cfg.Frontend.elasticSearch.Search(query, search.cfg.Frontend.cfg.PerPage, page)
	if err != nil {
		search.cfg.Frontend.Log.Panic(err)
	}

	// API like a bauss
	if r.FormValue("f") == "json" {
		output, err := json.Marshal(resp.Hits)
		if err != nil {
			search.cfg.Frontend.Log.Panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(output)
		if err != nil {
			search.cfg.Frontend.Log.Panic(err)
		}

		return
	}

	var results []Result
	for _, qr := range resp.Hits.Hits {
		raw, err := qr.Source.MarshalJSON()
		if err != nil {
			search.cfg.Frontend.Log.Panic(err)
		}

		var result Result
		err = json.Unmarshal(raw, &result)
		if err != nil {
			search.cfg.Frontend.Log.Panic(err)
		}

		result.HumanSize = humanize.Bytes(result.Size)
		results = append(results, result)
	}

	search.tmpl.ExecuteWriter(pongo2.Context{
		"query": query,

		"page":     page,
		"frompage": search.cfg.Frontend.cfg.PerPage * page,
		"maxpages": resp.Hits.Total / search.cfg.Frontend.cfg.PerPage,
		"prevpage": search.getPageLink(page-1, r.URL),
		"nextpage": search.getPageLink(page+1, r.URL),

		"elapsed":  time.Since(start) / time.Millisecond,
		"response": resp,
		"results":  results,
	}, w)
}

func (search *Search) getPageLink(page int, inURL *url.URL) (outURL string) {
	if page < 1 {
		page = 0
	}

	qry := inURL.Query()
	qry.Set("p", strconv.Itoa(page))
	inURL.RawQuery = qry.Encode()

	outURL = inURL.String()

	return
}
