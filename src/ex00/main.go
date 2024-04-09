package main

import (
	"client"
	"log"
	"strings"
)

const mapping = `
{
	"mappings": {
		"properties": {
			"name": {
				"type": "text"
			},
			"address": {
				"type": "text"
			},
			"phone": {
				"type": "text"
			},
			"location": {
				"type": "geo_point"
			}
		}
	}
}`

func main() {
	log.SetFlags(log.Lshortfile)

	client, err := client.CreateClient()
	if err != nil {
		log.Fatalln(err)
	}

	response, err := client.Indices.Create(
		"places",
		client.Indices.Create.WithBody(strings.NewReader(mapping)),
	)
	if err != nil {
		log.Fatalln(err)
	} else if response.StatusCode == 400 {
		log.Fatalln(response.String())
	}
}
