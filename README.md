# torture
Multi FTP crawler and file search. Written for CCC events.

torture exists of two seperate components: The crawler and the frontend. Both are interconnected via the Elasticsearch backend.

## Setup
It is always a good idea to setup a [GOPATH](https://golang.org/doc/code.html#GOPATH).

Dependencies are managed using [dep](https://github.com/golang/dep).

### General
1. Install and setup [Elasticsearch](https://www.elastic.co/products/elasticsearch)
1. Go into `crawler/` and run `go build`
1. Go into `frontend/` and run `go build`

### Crawler
1. See [crawler/README.md](crawler/README.md)

### Frontend
1. Run the frontend

### Docker
Setting up elasticsearch, crawler and frontend is significantly easier using Docker:

	cp crawler/config.json .
	docker-compose up -d

Reload the crawler's config:

	docker-compose kill -s USR1 crawler

## Authors
See [AUTHORS](AUTHORS). Do not forget to add yourself!

## License
See [LICENSE](LICENSE)
