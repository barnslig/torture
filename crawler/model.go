package main

import (
	"github.com/barnslig/torture/lib/elastic"
	"log"
	"time"
)

type ModelFileServerEntry struct {
	Url  string
	Path string
}

type ModelFileEntry struct {
	Filename string
	Size     int64
	MimeType string
	ModTime  time.Time
	LastSeen time.Time
	Servers  []ModelFileServerEntry
}

type hash map[string]interface{}

type Model struct {
	Host string
}

func CreateModel(host string) (model *Model, err error) {
	model = &Model{
		Host: host,
	}

	// Create mapping and index
	_, err = elastic.Request("PUT", elastic.URL(model.Host, "/torture"), hash{
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
				"properties": hash{
					"Filename": hash{
						"type": "keyword",
					},
					"Size": hash{
						"type": "long",
					},
					"MimeType": hash{
						"type": "keyword",
					},
					"ModTime": hash{
						"type": "date",
					},
					"LastSeen": hash{
						"type": "date",
					},
					"Servers": hash{
						"properties": hash{
							"Url": hash{
								"type": "keyword",
							},
							"Path": hash{
								"type":     "text",
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

func (model *Model) AddFileEntry(file ModelFileEntry) (err error) {
	// Check if the file entry already exists, then count up and/or append to
	// the servers array
	rawRes, err := elastic.Request("GET", elastic.URL(model.Host, "/torture/file/_search"), hash{
		"query": hash{
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
	})
	if err != nil {
		log.Println("find file error")
		return
	}

	res, err := elastic.ParseResponse(rawRes)
	if err != nil {
		log.Printf("find file parse error %s\n", rawRes)
		return
	}

	if res.Hits.Total > 0 {
		entry := &res.Hits.Hits[0]

		var updateRes []byte
		updateRes, err = elastic.Request("POST", elastic.URL(model.Host, "/torture/file/"+entry.Id+"/_update"), hash{
			"script": hash{
				"source": "ctx._source.LastSeen = params.LastSeen; if(!ctx._source.Servers.contains(params.Server)) { ctx._source.Servers.add(params.Server) }",
				"lang":   "painless",
				"params": hash{
					"LastSeen": time.Now(),
					"Server": hash{
						"Url":  file.Servers[0].Url,
						"Path": file.Servers[0].Path,
					},
				},
			},
		})

		if err != nil {
			log.Printf("%s\n", updateRes)
		}

		return
	}

	// If the file entry does not already exist, create it
	file.LastSeen = time.Now()
	_, err = elastic.Request("POST", elastic.URL(model.Host, "/torture/file"), file)

	return
}
