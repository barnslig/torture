package main

import (
	"encoding/json"
	"github.com/barnslig/torture/lib/elastic"
	"path"
)

type FtpEntry struct {
	Url  string
	Path string
}

type FileEntry struct {
	Filename string
	Size     uint64
	Servers  []FtpEntry
}

type ElasticSearch struct {
	url     string
	crawler *Crawler
}

type hash map[string]interface{}

func CreateElasticSearch(host string, cr *Crawler) (es *ElasticSearch, err error) {
	es = &ElasticSearch{
		crawler: cr,
		url:     host,
	}
	return
}

func (es *ElasticSearch) CreateMappingAndIndex() (err error) {
	_, err = elastic.Request("PUT", elastic.URL(es.url, "/torture"), hash{
		"settings": hash{
			"analysis": hash{
				"analyzer": hash{
					"filename": hash{
						"type":      "custom",
						"tokenizer": "filename",
						"filter":    []string{"lowercase"},
					},
				},
				"tokenizer": hash{
					"filename": hash{
						"type":    "pattern",
						"pattern": "[^\\p{L}\\d]+",
					},
				},
			},
		},
		"mappings": hash{
			"file": hash{
				"_timestamp": hash{
					"enabled": true,
				},
				"properties": hash{
					"Filename": hash{
						"type":  "string",
						"index": "not_analyzed",
					},
					"Size": hash{
						"type":  "double",
						"index": "not_analyzed",
					},
					"Servers": hash{
						"properties": hash{
							"Url": hash{
								"type":  "string",
								"index": "not_analyzed",
							},
							"Path": hash{
								"type":     "string",
								"analyzer": "filename",
							},
						},
					},
				},
			},
		},
	})

	// Ignore index_already_exists_exception
	if err != nil && err.Error() == "index_already_exists_exception" {
		err = nil
	}

	return
}

// Look up a FileEntry by filename and size and
func (es *ElasticSearch) GetFileEntry(file FileEntry) (entry *elastic.Hit, err error) {
	data, err := elastic.Request("GET", elastic.URL(es.url, "/torture/file/_search"), hash{
		"query": hash{
			"filtered": hash{
				"filter": hash{
					"bool": hash{
						"must": []hash{
							hash{
								"term": hash{
									"Filename": file.Filename,
								},
							},
							hash{
								"term": hash{
									"Size": file.Size,
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return
	}

	res := elastic.Result{}
	err = json.Unmarshal(data, &res)
	if err != nil {
		return
	}

	if res.Hits.Total > 0 {
		entry = &res.Hits.Hits[0]
	}

	return
}

func (es *ElasticSearch) AddFileEntry(file FileEntry) (res []byte, err error) {
	// check if the FileEntry already exists
	entry, err := es.GetFileEntry(file)
	if err != nil {
		return
	}

	// Entry already exists; Add/update a server entry
	if entry != nil {
		es.crawler.Log.Printf("%s: EXISTS %s", file.Servers[0].Url, file.Servers[0].Path)

		res, err = elastic.Request("POST", elastic.URL(es.url, path.Join("/torture/file/", entry.Id, "/_update")), hash{
			"script": "ctx._source.Servers.contains(Server) ? (ctx.op = \"none\") : (ctx._source.Servers += Server)",
			"params": hash{
				"Server": hash{
					"Url":  file.Servers[0].Url,
					"Path": file.Servers[0].Path,
				},
			},
		})
		return
	}

	es.crawler.Log.Printf("%s: %s", file.Servers[0].Url, file.Servers[0].Path)

	// Entry does not exist; Index it
	res, err = elastic.Request("POST", elastic.URL(es.url, "/torture/file"), file)
	if err != nil {
		return
	}

	return
}
