package elastic

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"log"
)

type Hit struct {
	Id     string           `json:"_id"`
	Score  float32          `json:"_score"`
	Source *json.RawMessage `json:"_source"`
}

type Hits struct {
	Total int   `json:"total"`
	Hits  []Hit `json:"hits"`
}

type Result struct {
	Hits         Hits             `json:"hits"`
	Aggregations *json.RawMessage `json:"aggregations"`
}

type Error struct {
	Type string `json:"type"`
}

// Add a path to a given host
// Example: URL("http://localhost:9200", "/torture") --> http://localhost:9200/torture
func URL(host string, path string) string {
	parsedHost, err := url.Parse(host)
	if err != nil {
		panic(err)
	}

	parsedHost.Path = path
	return parsedHost.String()
}

// Parse a response into the Result struct
func ParseResponse(data []byte) (result Result, err error) {
	err = json.Unmarshal(data, &result)
	return
}

func Request(method string, url string, payload interface{}) (data []byte, err error) {
	// Encode the JSON payload
	encPayload, err := json.Marshal(payload)
	if err != nil {
		return
	}

	// Create a new HTTP request
	req, err := http.NewRequest(method, url, bytes.NewReader(encPayload))
	if err != nil {
		return
	}

	req.Header.Add("Content-Type", "application/json")

	// Do the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Parse the response body into a []byte
	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// Check that we have no error
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return
	}

	// Otherwise try to parse the error
	outerMsg := make(map[string]*json.RawMessage)
	err = json.Unmarshal(data, &outerMsg)
	if err != nil {
		return
	}

	esError := Error{}
	err = json.Unmarshal(*outerMsg["error"], &esError)
	if err != nil {
		log.Printf("%s\n", *outerMsg["error"])
		return
	}

	err = errors.New(esError.Type)
	return
}
