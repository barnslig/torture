package main

import (
	"gopkg.in/flosch/pongo2.v3"
	"io"
	"log"
	"net/http"
)

type FrontendConfig struct {
	HttpListen    string
	ElasticServer string
	LogOutput     io.Writer
	PerPage       int
}

type Frontend struct {
	cfg           FrontendConfig
	elasticSearch *ElasticSearch
	templates     *pongo2.TemplateSet

	Log *log.Logger
}

func CreateFrontend(cfg FrontendConfig) (frontend *Frontend, err error) {
	frontend = &Frontend{cfg: cfg}

	// Create logger
	frontend.Log = log.New(frontend.cfg.LogOutput, "frontend: ", log.Ldate|log.Lshortfile)

	// Create an ElasticSearch connection
	frontend.elasticSearch, err = CreateElasticSearch(frontend.cfg.ElasticServer)
	if err != nil {
		return
	}

	// Create a pongo2 template set
	frontend.templates = pongo2.NewSet("torture")
	frontend.templates.SetBaseDirectory("templates")

	// Sub-Apps
	search, err := CreateSearch(SearchConfig{
		Frontend: frontend,
	})
	if err != nil {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/s", search.Handler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/", http.RedirectHandler("/s", 301))

	log.Fatal(http.ListenAndServe(frontend.cfg.HttpListen, mux))

	return
}
