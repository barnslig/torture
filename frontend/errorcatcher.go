package main

import (
	"encoding/json"
	"github.com/flosch/pongo2"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type JsonError struct {
	Error string `json:"error"`
}

type ErrorCatcherConfig struct {
	Frontend *Frontend
}

type ErrorCatcher struct {
	cfg  ErrorCatcherConfig
	tmpl *pongo2.Template
}

func CreateErrorCatcher(cfg ErrorCatcherConfig) (errorCatcher *ErrorCatcher, err error) {
	errorCatcher = &ErrorCatcher{cfg: cfg}

	// Load the error page template
	errorCatcher.tmpl, err = errorCatcher.cfg.Frontend.templates.FromFile("error.tmpl")
	if err != nil {
		return
	}

	return
}

func (errorCatcher *ErrorCatcher) Handler(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

		// Catch errors in the following code, log them and return a HTTP 500
		defer func() {
			if err, ok := recover().(error); ok {
				errorCatcher.cfg.Frontend.Log.Println(err)

				w.WriteHeader(http.StatusInternalServerError)

				/* Parse GET parameters
				 * format: Format. Default is HTML, currently supported options: "json"
				 */
				format := r.FormValue("format")

				if format == "json" {
					output, err := json.Marshal(JsonError{
						Error: err.Error(),
					})
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

				errorCatcher.tmpl.ExecuteWriter(pongo2.Context{
					"msg": err.Error(),
				}, w)
			}
		}()

		h(w, r, params)
	}
}
