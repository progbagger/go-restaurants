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
	"sync"
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
	if err != nil {
		return nil, fmt.Errorf("can't delete index \"%s\", skipping", index)
	}
	if response.IsError() {
		return response, nil
	}

	log.Printf("Deleted previously created index \"%s\"", index)
	response.Body.Close()

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

func insertRecord(indexer *esutil.BulkIndexer, record common.RestaurantRecord, id uint64) error {
	marshalizedRecord, err := json.Marshal(record)
	if err != nil {
		return err
	}

	err = (*indexer).Add(
		context.Background(),
		esutil.BulkIndexerItem{
			Action:     "index",
			DocumentID: fmt.Sprint(id),
			Body:       bytes.NewReader(marshalizedRecord),
		},
	)

	return err
}

func readAndInsertRecords(indexer *esutil.BulkIndexer, csvReader *csv.Reader) (readedRecords uint64, succesfullyInsertedRecords uint64) {
	succesfullyInsertedRecords, readedRecords = 0, 0

	isHeaderReaded := false
	waitGroup := sync.WaitGroup{}
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
		waitGroup.Add(1)

		// asynchronously add records to the index
		go func(currentId uint64) {
			defer waitGroup.Done()

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

			err = insertRecord(
				indexer,

				common.RestaurantRecord{
					Name:    record[1],
					Address: record[2],
					Phone:   record[3],
					Location: common.Location{
						Longitude: lon,
						Latitude:  lat,
					},
				},

				currentId,
			)
			if err != nil {
				log.Printf("%s: %s\n", errorMessage, err)
				return
			}
			atomic.AddUint64(&succesfullyInsertedRecords, 1)
		}(readedRecords)
	}

	waitGroup.Wait()

	return readedRecords, succesfullyInsertedRecords
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

	readedRecords, succesfullyInsertedRecords := readAndInsertRecords(&bulkIndexer, csvReader)

	if err := bulkIndexer.Close(context.Background()); err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("Succesfully readed records: %d\nSuccesfully inserted records: %d\n",
		readedRecords,
		succesfullyInsertedRecords,
	)
}
