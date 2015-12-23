package main

import (
	"encoding/json"
	"github.com/barnslig/torture/lib/elastic"
)

type ElasticSearch struct {
	url string
}

type hash map[string]interface{}

func CreateElasticSearch(host string) (es *ElasticSearch, err error) {
	es = &ElasticSearch{url: host}
	return
}

func (es *ElasticSearch) Search(query string, filters Filter, perPage int, page int) (result elastic.Result, err error) {
	matchQ := hash{
		"match": hash{
			"Servers.Path": hash{
				"query":     query,
				"fuzziness": 1,
			},
		},
	}

	filterQ := make(hash)

	// Filter: Files smaller than 100B
	if filters.SmallFiles {
		filterQ["range"] = hash{
			"Size": hash{
				"gte": 100,
			},
		}
	}

	data, err := elastic.Request("GET", elastic.URL(es.url, "/torture/file/_search"), hash{
		"size": perPage,
		"from": perPage * page,
		"query": hash{
			"filtered": hash{
				"query":  matchQ,
				"filter": filterQ,
			},
		},
	})
	if err != nil {
		return
	}

	result = elastic.Result{}
	err = json.Unmarshal(data, &result)

	return
}
