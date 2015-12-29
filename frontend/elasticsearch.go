package main

import (
	"encoding/json"
	"fmt"
	"github.com/barnslig/torture/lib/elastic"
	"github.com/dustin/go-humanize"
	"regexp"
	"strings"
)

type ElasticSearch struct {
	url string
}

type hash map[string]interface{}

func CreateElasticSearch(host string) (es *ElasticSearch, err error) {
	es = &ElasticSearch{url: host}
	return
}

func (es *ElasticSearch) Search(stmt Statement, perPage int, page int) (result elastic.Result, err error) {
	query := strings.Join(stmt.Phrases, " ")

	matchQ := hash{
		"function_score": hash{
			"query": hash{
				"query_string": hash{
					"default_field": "Servers.Path",
					"default_operator": "AND",
					"query": query,
				},
			},
			"field_value_factor": hash{
				"field":    "Size",
				"missing":  1,
				"modifier": "log1p",
				"factor":   0.000000001, // Increase Bytes to Gigabytes
			},
		},
	}

	filterQ := []hash{}
	for _, treat := range stmt.Treats {

		// Filter for file extensions, e.g. extension:pdf or extension!mkv
		if treat.Key == keyExtension {
			regex := ""
			switch treat.Operator {
			case EQUALS:
				regex = fmt.Sprintf(`.+.%s`, ExtractRegexSave(treat.Value))
				break
			case NOT:
				regex = fmt.Sprintf(`@&~(.+.%s)`, ExtractRegexSave(treat.Value))
				break
			default:
				continue
			}

			filterQ = append(filterQ, hash{
				"regexp": hash{
					"Filename": regex,
				},
			})
		}

		// Filter for size, e.g. size>1gb or size<20mb
		if treat.Key == keySize {
			// Try to parse the given size
			size, err := humanize.ParseBytes(treat.Value)
			if err != nil {
				continue
			}

			rangeQ := hash{}
			switch treat.Operator {
			case LTE:
				rangeQ = hash{
					"lte": size,
				}
				break
			case GTE:
				rangeQ = hash{
					"gte": size,
				}
			default:
				continue
			}

			filterQ = append(filterQ, hash{
				"range": hash{
					"Size": rangeQ,
				},
			})
		}

	}

	data, err := elastic.Request("GET", elastic.URL(es.url, "/torture/file/_search"), hash{
		"size": perPage,
		"from": perPage * page,
		"query": hash{
			"filtered": hash{
				"query": matchQ,
				"filter": hash{
					"and": hash{
						"filters": filterQ,
					},
				},
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

// Extract only regex-save characters from a string so we can Sprintf
func ExtractRegexSave(src string) string {
	r := regexp.MustCompile(`\w+`)
	return strings.Join(r.FindAllString(src, -1), "")
}
