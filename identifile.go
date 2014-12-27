package main

import (
	"fmt"
	elastigo "github.com/barnslig/elastigo/lib"
	"log"
)

var (
	es_conn *elastigo.Conn
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

type hash map[string]interface{}

func initElastics(host string) {
	es_conn = elastigo.NewConn()
	es_conn.Domain = host

	es_conn.CreateIndex("torture")

	// enable timestamps
	err := es_conn.PutMapping("torture", "file", FileEntry{}, elastigo.MappingOptions{
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
	if err != nil {
		log.Fatal(err)
	}
}

func addToElastic(file FileEntry) {
	// look up the file by the name
	searchJson := fmt.Sprintf(`{
		"query": {
			"filtered": {
				"filter": {
					"bool": {
						"must": [
							{"term": {
								"Filename": "%s"
							}},
							{"term": {
								"Size": %d
							}}
						]
					}
				}
			}
		}
	}`, file.Filename, file.Size)
	query_response, err := es_conn.Search("torture", "file", nil, searchJson)
	if err != nil {
		log.Print(err)
		return
	}
	if query_response.Hits.Len() > 0 {
		log.Println(query_response.Hits.Hits[0].Id)

		_, err := es_conn.Update("torture", "file", query_response.Hits.Hits[0].Id, nil, hash{
			"script": "ctx._source.Servers.contains(Server) ? (ctx.op = \"none\") : (ctx._source.Servers += Server)",
			"params": hash{
				"Server": hash{
					"Url":  file.Servers[0].Url,
					"Path": file.Servers[0].Path,
				},
			},
		})
		if err != nil {
			log.Print(err)
		}
	} else {
		_, err := es_conn.Index("torture", "file", "", nil, file)
		if err != nil {
			log.Print(err)
		}
	}
}
