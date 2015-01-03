package main

import (
	elastigo "github.com/barnslig/elastigo/lib"
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
	Conn    *elastigo.Conn
	crawler *Crawler
}

type hash map[string]interface{}

func CreateElasticSearch(host string, cr *Crawler) (es *ElasticSearch, err error) {
	es = &ElasticSearch{crawler: cr}

	es.Conn = elastigo.NewConn()
	es.Conn.Domain = host
	return
}

func (es *ElasticSearch) CreateMappingAndIndex() (err error) {
	_, err = es.Conn.CreateIndex("torture")
	if err != nil {
		return
	}

	// enable timestamps + do not analyze Filenames so we can do exact matches
	err = es.Conn.PutMapping("torture", "file", FileEntry{}, elastigo.MappingOptions{
		Timestamp: elastigo.TimestampOptions{
			Enabled: true,
			Store:   true,
		},
		Properties: hash{
			"Filename": hash{
				"type":  "string",
				"index": "not_analyzed",
			},
		},
	})

	return
}

// Look up a FileEntry by filename and size and
func (es *ElasticSearch) GetFileEntry(file FileEntry) (entry *elastigo.Hit, err error) {
	searchQ := hash{
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
	}
	response, err := es.Conn.Search("torture", "file", nil, searchQ)
	if err != nil {
		return
	}

	if response.Hits.Len() > 0 {
		entry = &response.Hits.Hits[0]
	}
	return
}

func (es *ElasticSearch) AddFileEntry(file FileEntry) (res elastigo.BaseResponse, err error) {
	// check if the FileEntry already exists
	entry, err := es.GetFileEntry(file)
	if err != nil {
		return
	}

	// Entry already exists; Add/update a server entry
	if entry != nil {
		es.crawler.Log.Printf("%s: EXISTS %s", file.Servers[0].Url, file.Servers[0].Path)

		res, err = es.Conn.Update("torture", "file", entry.Id, nil, hash{
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
	res, err = es.Conn.Index("torture", "file", "", nil, file)
	return
}
