package main

import (
	"common"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type Store interface {
	// returns a list of items,
	// a total number of hits and (or) an error in case of one
	GetPlaces(limit int, offset int) ([]common.Place, int, error)
}

type ElasticPaginator struct {
	Client *elasticsearch.Client
	Index  string
}

type ElasticPaginatorResponseResult struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`

		Hits []struct {
			ID     string       `json:"_id"`
			Score  float64      `json:"_score"`
			Source common.Place `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func (paginator *ElasticPaginator) GetPlaces(limit int, offset int) ([]common.Place, int, error) {
	places := make([]common.Place, 0, limit)
	totalHits := 0

	for i := 0; i <= limit; i += 10_000 {
		currentLimit := 10_000
		if limit-i < 10_000 {
			currentLimit = limit - i
		}

		request := esapi.SearchRequest{
			Index: []string{paginator.Index},
			From:  &i,
			Size:  &currentLimit,
			Sort:  []string{},
		}

		response, err := request.Do(context.Background(), paginator.Client)
		if err != nil {
			return nil, 0, err
		}
		defer response.Body.Close()

		if response.IsError() {
			return nil, 0, fmt.Errorf("%s", response)
		}

		var result ElasticPaginatorResponseResult
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			return nil, 0, err
		}

		totalHits += result.Hits.Total.Value

		for _, place := range result.Hits.Hits {
			places = append(places, place.Source)
		}
	}

	return places, totalHits, nil
}
