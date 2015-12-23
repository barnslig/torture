# torture
Multi FTP crawler and file search. Written for CCC events.

torture exists of two seperate components: The crawler and the frontend. Both are interconnected via the Elasticsearch backend.

## Setup
It is always a good idea to setup a [GOPATH](https://golang.org/doc/code.html#GOPATH).

For dependency version management, gopm is used. [Please install gopm before continuing](https://github.com/gpmgo/gopm#installation)!

### General
1. Install and setup [Elasticsearch](https://www.elastic.co/products/elasticsearch)
2. Go into `crawler/` and run `gopm get; gopm build`
3. Go into `frontend/` and run `gopm get; gopm build`

### Crawler
1. Create a `servers.txt` somewhere. Example is given inside `crawler/`.
2. You need to add `script.engine.groovy.inline.update: true` to your `elasticsearch.yml`
3. Run the crawler

### Frontend
1. Run the frontend

## Authors
See [AUTHORS](AUTHORS). Do not forget to add yourself!

## License
See [LICENSE](LICENSE)
