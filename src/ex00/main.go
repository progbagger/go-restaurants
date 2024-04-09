package main

import (
	"bytes"
	"client"
	"common"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8/esutil"
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

func recreateIndex(client *elasticsearch.Client, index, mapping string) (*esapi.Response, error) {
	response, err := client.Indices.Delete([]string{index}, client.Indices.Delete.WithIgnoreUnavailable(true))
	if err != nil || response.IsError() {
		log.Printf("Can't delete index \"%s\". Skipping...\n", index)
	} else if response.IsError() {
		log.Printf("Deleted previously created index \"%s\"", index)
	} else {
		response.Body.Close()
	}

	response, err = client.Indices.Create(
		index,
		client.Indices.Create.WithBody(strings.NewReader(mapping)),
	)
	if err != nil {
		return nil, err
	}
	if response.IsError() {
		return response, nil
	}
	response.Body.Close()
	log.Printf("Created index \"%s\"\n", index)

	return nil, nil
}

func insertDocument(indexer *esutil.BulkIndexer, record common.RestaurantRecord) error {
	marshalizedRecord, err := json.Marshal(record)
	if err != nil {
		return err
	}

	err = (*indexer).Add(
		context.Background(),
		esutil.BulkIndexerItem{
			Action: "index",
			Body:   bytes.NewReader(marshalizedRecord),

			OnSuccess: func(ctx context.Context, bii esutil.BulkIndexerItem, biri esutil.BulkIndexerResponseItem) {
				log.Println("Succesfully inserted record", record)
			},
		},
	)

	return err
}

func main() {
	log.SetFlags(log.Lshortfile)

	// creating CSV reader
	restaurantsFile, err := os.Open("../../materials/data.csv")
	if err != nil {
		log.Fatalln()
	}
	defer restaurantsFile.Close()

	csvReader := csv.NewReader(restaurantsFile)
	csvReader.Comma = '\t'
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true

	client, err := client.CreateClient()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Created elasticsearch client")

	response, err := recreateIndex(client, "places", mapping)
	if err != nil {
		log.Fatalf("Can't create index \"places\": %s", err)
	} else if response != nil && response.IsError() {
		log.Fatalf("Can't create index \"places\": %s", response)
	}

	// creating indexer
	bulkIndexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:  "places",
		Client: client,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// reading CSV and indexing
	var succesfullyInsertedRecords uint64 = 0
	var readedRecords uint64 = 0

	isHeaderReaded := false
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalln(err)
		}

		// skip header
		if !isHeaderReaded {
			isHeaderReaded = true
			continue
		}

		readedRecords++

		// asynchronously add records to the index
		go func() {
			errorMessage := "Couldn't insert record"

			lon, err := strconv.ParseFloat(record[4], 64)
			if err != nil {
				log.Printf("%s: %s\n", errorMessage, err)
				return
			}

			lat, err := strconv.ParseFloat(record[5], 64)
			if err != nil {
				log.Printf("%s: %s\n", errorMessage, err)
				return
			}

			err = insertDocument(&bulkIndexer, common.RestaurantRecord{
				Name:    record[1],
				Address: record[2],
				Phone:   record[3],
				Location: common.Location{
					Longitude: lon,
					Latitude:  lat,
				},
			})
			if err != nil {
				log.Printf("%s: %s\n", errorMessage, err)
				atomic.AddUint64(&succesfullyInsertedRecords, 1)
				return
			}
		}()
	}

	if err := bulkIndexer.Close(context.Background()); err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("Succesfully readed records: %d\nSuccesfully inserted records: %d\n",
		readedRecords,
		succesfullyInsertedRecords,
	)
}
