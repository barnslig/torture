# Contentment Bot Usage Instructions
This nearly complete rewrite of the old bot has some nice new features:
- Flexible crawling backend support: *Make it easy to implement additional protocols like HTTP, NFS, …*
- HTTP crawling backend: *Finally get rid of old and busted FTP!*
- Crawling rate limiting: *Limit the amount of requests so fileservers can breath again*
- Last Seen Timestamp: *Files which have not been seen for a long time can be prioritized lower in results*
- Elasticsearch >5 support
- Dependency management: *Get rid of shitty gopm*

## Setup

1. Have a working [GOPATH](https://golang.org/doc/code.html#GOPATH).
2. Modify the `config.json` file
3. `go build`
4. `./crawler`

Dependencies are managed using [govendor](https://github.com/kardianos/govendor).

## Implementing new protocols

1. Create a new crawler implementing the `Crawler` interface (see crawler.go)
2. Make sure your crawler supports:
   * robots.txt parsing (see existing crawlers for examples)
   * rate limiting
3. Add initialization of your crawler to the switch in crawler.go

## FTP Crawler Config Options

* entry
  * string
  * required
  * no default
  * URI of the entry point, might optionally be with path, port, username, password e.g. ftp://hans:geheim@foo:21/bar/baz/
  * Default username and password: anonymous:anonymous
* turnDelay
  * integer
  * optional
  * default: 10
  * Amount of seconds to wait after the complete server got crawled before starting again
* maxRequestPerSecond
  * integer
  * optional
  * default: 0
  * Maximum amount of requests per second. 0 means unlimited
* robotName
  * string
  * optional
  * default: TortureBot
  * Name of the robot to check against in /robots.txt
* obeyRobotsTxt
  * boolean
  * optional
  * default: true
  * Whether to check pathes against /robots.txt

## HTTP Crawler Config Options

* entry
  * string
  * required
  * no default
  * URI of the entry point, might optionally be with path, port, username, password e.g. http://hans:geheim@foo:21/bar/baz/
* turnDelay
  * integer
  * optional
  * default: 10
  * Amount of seconds to wait after the complete server got crawled before starting again
* maxRequestPerSecond
  * integer
  * optional
  * default: 0
  * Maximum amount of requests per second. 0 means unlimited
* robotName
  * string
  * optional
  * default: TortureBot
  * Name of the robot to check against in /robots.txt
* obeyRobotsTxt
  * boolean
  * optional
  * default: true
  * Whether to check pathes against /robots.txt
* maxBodySize
  * integer
  * optional
  * default: 1 Megabyte (10000000)
  * Maximum downloaded body size. In order to search for link, we have to download pages.
* maxPathDepth
  * integer
  * optional
  * default: 20
  * Maximum path depth. Used to "catch" symlink loops
