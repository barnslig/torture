package main

import (
	"github.com/flosch/pongo2"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type HelpConfig struct {
	Frontend *Frontend
}

type Help struct {
	cfg  HelpConfig
	tmpl *pongo2.Template
}

func CreateHelp(cfg HelpConfig) (help *Help, err error) {
	help = &Help{cfg: cfg}

	// Load the error page template
	help.tmpl, err = help.cfg.Frontend.templates.FromFile("help.tmpl")
	if err != nil {
		return
	}

	return
}

func (help *Help) Handler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	help.tmpl.ExecuteWriter(pongo2.Context{}, w)
}
