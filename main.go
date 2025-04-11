package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func main() {

	var url, jsonpath, header string
	var dev bool

	const (
		// per https://cloud.google.com/docs/authentication/rest#impersonated-sa
		DEFAULT_URL      = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token"
		DEFAULT_JSONPATH = "access_token"
		DEFAULT_HEADER   = "Metadata-Flavor: Google"
	)

	flag.StringVar(&url, "url", DEFAULT_URL, "Url to get token credential")
	flag.StringVar(&jsonpath, "jsonpath", DEFAULT_JSONPATH, "Path to token in the url json content")
	flag.StringVar(&header, "header", DEFAULT_HEADER, "Any header to send")
	flag.BoolVar(&dev, "dev", false, "url override to use local serving path for dev")
	flag.Parse()

	if dev {
		// use this by running `python3 -m http.server -d examples`
		url = "http://127.0.0.1:8000/meta-server-response.json"
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v\n", err)
	}
	req.Header.Set("Accept", "application/json")

	if len(header) > 0 {
		parts := strings.SplitN(header, ": ", 2)
		if len(parts) != 2 {
			log.Fatalf("invalid header format: expected 'Key: Value', got %q", header)
		}
		req.Header.Set(parts[0], parts[1])
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error fetching URL %s: %v", url, err)
	}
	defer resp.Body.Close() // Ensure the response body is closed

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error: Received non-200 status code %d from %s", resp.StatusCode, url)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	var sourceData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &sourceData); err != nil {
		log.Fatalf("Error unmarshaling source JSON: %v", err)
	}

	extractedValue, found := sourceData[jsonpath]
	if !found {
		log.Fatalf("Error: Key '%s' not found in the JSON response from %s", jsonpath, url)
	}

	type response struct {
		ApiVersion     string `json:"apiVersion"`
		ExecCredential string
		Status         map[string]string `json:"status"`
	}

	responseDoc := response{
		ApiVersion:     "client.authentication.k8s.io/v1beta1",
		ExecCredential: "ExecCredential",
		Status: map[string]string{
			"token": extractedValue.(string),
		},
	}
	outputBytes, err := json.Marshal(responseDoc)
	if err != nil {
		log.Fatalf("Error marshaling output JSON: %v", err)
	}

	fmt.Println(string(outputBytes))
}
