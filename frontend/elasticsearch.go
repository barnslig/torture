package main

import (
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

		// Filter for media types, e.g. type:video, type:audio
		if treat.Key == keyType {
			endingsRegex := "*"
			switch treat.Value {
			case "video":
				endingsRegex = "(webm|mkv|flv|vob|ogv|avi|mov|wmv|mp4|mpg|mpeg|m4v|3gp|mts)"
				break
			case "audio":
				endingsRegex = "(aac|aiff|amr|flac|m4a|mp3|ogg|oga|opus|wav|wma)"
				break
			case "image":
				endingsRegex = "(jpg|jpeg|tiff|gif|bmp|png|webp|psd|xcf|svg|ai)"
				break
			case "document":
				endingsRegex = "(epub|doc|docx|html|tex|ibooks|azw|mobi|pdf|txt|ps|rtf|xps|odt)"
				break
			default:
				continue
			}

			regex := ""
			switch treat.Operator {
			case EQUALS:
				regex = fmt.Sprintf(`.+.%s`, endingsRegex)
				break
			case NOT:
				regex = fmt.Sprintf(`@&~(.+.%s)`, endingsRegex)
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

	}

	data, err := elastic.Request("POST", elastic.URL(es.url, "/torture/file/_search"), hash{
		"size": perPage,
		"from": perPage * page,
		"query": hash{
			"bool": hash{
				"must": hash{
					"function_score": hash{
						"query": hash{
							"simple_query_string": hash{
								"fields":           []string{"Servers.Path"},
								"default_operator": "AND",
								"query":            query,
							},
						},
						"field_value_factor": hash{
							"field":    "Size",
							"missing":  1,
							"modifier": "log1p",
							"factor":   0.000000001, // Increase Bytes to Gigabytes
						},
					},
				},
				"filter": filterQ,
			},
		},
	})
	if err != nil {
		return
	}

	result, err = elastic.ParseResponse(data)

	return
}

// Extract only regex-save characters from a string so we can Sprintf
func ExtractRegexSave(src string) string {
	r := regexp.MustCompile(`\w+`)
	return strings.Join(r.FindAllString(src, -1), "")
}
