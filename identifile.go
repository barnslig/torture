package main

import (
	"log"
	elastigo "github.com/mattbaird/elastigo/lib"
)

var (
	es_conn *elastigo.Conn
)

type FileEntry struct {
	Server string
	Path string
	Size uint64
}

func initElastics(host string) {
	es_conn = elastigo.NewConn()
	es_conn.Domain = host

	es_conn.CreateIndex("torture")

	// enable timestamps
	err := es_conn.PutMapping("torture", "file", FileEntry{}, elastigo.MappingOptions{
		Timestamp: elastigo.TimestampOptions{
			Enabled: true,
			Store: true,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func addToElastic(file FileEntry) {
	_, err := es_conn.Index("torture", "file", "", nil, file)
	if err != nil {
		log.Print(err)
	}
}
