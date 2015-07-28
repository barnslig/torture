package main

import (
	elastigo "github.com/barnslig/elastigo/lib"
)

type ElasticSearch struct {
	Conn *elastigo.Conn
}

type hash map[string]interface{}

func CreateElasticSearch(host string) (es *ElasticSearch, err error) {
	es = &ElasticSearch{}

	es.Conn = elastigo.NewConn()
	es.Conn.Domain = host

	return
}

func (es *ElasticSearch) Search(query string, perPage int, page int) (resp elastigo.SearchResult, err error) {
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

	resp, err = es.Conn.Search("torture", "file", hash{
		"size": perPage,
		"from": perPage * page,
	}, searchQ)

	return
}
