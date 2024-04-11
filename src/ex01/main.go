package main

import (
	"args"
	"db"
	"fmt"
	"log"
	"os"
)

func main() {
	log.SetFlags(log.Lshortfile)

	parsedArgs, _, err := args.ParseArgs(
		args.Arg{
			Name:         "cacert",
			Description:  "PAth to the http_ca.crt file",
			DefaultValue: "",
			Required:     true,
		},
	)
	if err != nil {
		log.Fatalln(err)
	}

	CACert, err := os.ReadFile(parsedArgs["cacert"].(string))
	if err != nil {
		log.Fatalln(err)
	}

	client, err := db.CreateClient(CACert)
	if err != nil {
		log.Fatalln(err)
	}

	paginator := ElasticPaginator{Client: client, Index: "places"}
	places, hits, err := paginator.GetPlaces(15_000, 13_640)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("hits: %d\n", hits)
	fmt.Printf("len: %d\n", len(places))

	for _, place := range places {
		fmt.Println(place)
	}
}
