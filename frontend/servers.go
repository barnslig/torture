package main

import (
	"encoding/json"
	"github.com/barnslig/torture/lib/elastic"
	"github.com/dustin/go-humanize"
	"github.com/flosch/pongo2"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"time"
)

// JSON data structures
type ServersValue struct {
	Value json.Number `json:"value"`
}

type ServersBucket struct {
	Url            string       `json:"key"`
	FileCount      json.Number  `json:"doc_count"`
	LatestFileTime ServersValue `json:"latest_file"`
	FullSize       ServersValue `json:"full_size"`
}

type ServersByUrl struct {
	Buckets []ServersBucket `json:"buckets"`
}

type ServersAggregations struct {
	ByUrl ServersByUrl `json:"by_url"`
}

// view data structure
type ServersServer struct {
	Url            string
	FileCount      uint64
	FullSize       string
	LatestFileTime string
}

// config and instance structures
type ServersConfig struct {
	Frontend *Frontend
}

type Servers struct {
	cfg  ServersConfig
	tmpl *pongo2.Template
}

func CreateServers(cfg ServersConfig) (servers *Servers, err error) {
	servers = &Servers{cfg: cfg}

	// Load the servers page template
	servers.tmpl, err = servers.cfg.Frontend.templates.FromFile("servers.tmpl")
	if err != nil {
		return
	}

	return
}

func (servers *Servers) Handler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	/* Parse GET parameters
	 * format: Format. Default is HTML, currently supported options: "json"
	 */
	format := r.FormValue("format")

	query := hash{
		"size": 0,
		"aggs": hash{
			"by_url": hash{
				"terms": hash{
					"size": 2147483647,
					"field": "Servers.Url",
				},
				"aggs": hash{
					"full_size": hash{
						"sum": hash{
							"field": "Size",
						},
					},
					"latest_file": hash{
						"max": hash{
							"field": "LastSeen",
						},
					},
				},
			},
		},
	}

	data, err := elastic.Request("GET", elastic.URL(servers.cfg.Frontend.elasticSearch.url, "/torture/file/_search"), query)
	if err != nil {
		panic(err)
	}

	// Unmarshal raw results
	result := elastic.Result{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		panic(err)
	}

	// Unmarshal specific aggregations
	aggs := ServersAggregations{}
	err = json.Unmarshal(*result.Aggregations, &aggs)
	if err != nil {
		panic(err)
	}

	// Create a data structure for the view
	serversList := []ServersServer{}
	for _, server := range aggs.ByUrl.Buckets {
		sizeFloat, _ := server.FullSize.Value.Float64()
		fullSize := humanize.Bytes(uint64(sizeFloat))

		latestFileTimeFloat, _ := server.LatestFileTime.Value.Float64()
		latestFileTime := humanize.Time(time.Unix(int64(latestFileTimeFloat/1000), 0))

		fileCountFloat, _ := server.FileCount.Float64()
		fileCount := uint64(fileCountFloat)

		serversList = append(serversList, ServersServer{
			Url:            server.Url,
			FileCount:      fileCount,
			FullSize:       fullSize,
			LatestFileTime: latestFileTime,
		})
	}

	if format == "json" {
		output, err := json.Marshal(serversList)
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

	servers.tmpl.ExecuteWriter(pongo2.Context{
		"servers": serversList,
	}, w)
}
