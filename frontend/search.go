package main

import (
	"encoding/json"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/flosch/pongo2"
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

	// Catch errors in the following code, log them and return a HTTP 500
	defer func() {
		if err := recover(); err != nil {
			search.cfg.Frontend.Log.Println(err)
			http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		}
	}()

	/* Parse GET parameters
	 * q: Search query
	 * f: Format. Default is HTML, currently supported options: "json"
	 * p: Current page. Is zero if not a number
	 */
	query := r.FormValue("q")
	format := r.FormValue("f")
	page, err := strconv.Atoi(r.FormValue("p"))
	if err != nil {
		page = 0
	}

	// Do the actual search
	resp, err := search.cfg.Frontend.elasticSearch.Search(query, search.cfg.Frontend.cfg.PerPage, page)
	if err != nil {
		search.cfg.Frontend.Log.Panic(err)
	}

	// Format: JSON
	if format == "json" {
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

	// Format: HTML (default)
	var results []Result
	for _, qr := range resp.Hits.Hits {
		// Parse the search result into a Result struct
		var result Result
		err := unmarshalRawJson(qr.Source, &result)
		if err != nil {
			search.cfg.Frontend.Log.Panic(err)
		}

		// Humanize the file size
		result.HumanSize = humanize.Bytes(result.Size)
		results = append(results, result)
	}

	search.tmpl.ExecuteWriter(pongo2.Context{
		"query": query,

		"page":     page,
		"frompage": search.cfg.Frontend.cfg.PerPage * page,
		"maxpages": resp.Hits.Total / search.cfg.Frontend.cfg.PerPage,
		"prevpage": getPageLink(page-1, r.URL),
		"nextpage": getPageLink(page+1, r.URL),

		"elapsed":  time.Since(start) / time.Millisecond,
		"response": resp,
		"results":  results,
	}, w)
}

func getPageLink(page int, inURL *url.URL) (outURL string) {
	if page < 1 {
		page = 0
	}

	qry := inURL.Query()
	qry.Set("p", strconv.Itoa(page))
	inURL.RawQuery = qry.Encode()

	outURL = inURL.String()

	return
}

func unmarshalRawJson(input *json.RawMessage, output interface{}) (err error) {
	raw, err := input.MarshalJSON()
	if err != nil {
		return
	}

	err = json.Unmarshal(raw, &output)
	if err != nil {
		return
	}

	return
}
