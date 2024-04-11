package paginate

import (
	"common"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
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
			Sort   []any        `json:"sort"`
		} `json:"hits"`
	} `json:"hits"`
}

const query = `
{
	"size": %d,
	"sort": [
		%s
	]%s
}`

type SortParameter struct {
	Field      string
	Descending bool
}

func buildQuery(limit int, searchAfter []any, params []SortParameter) (string, error) {
	if limit < 0 {
		return "", fmt.Errorf("negative limit is not allowed")
	}

	if len(params) == 0 {
		return "", fmt.Errorf("empty sort parameters are not allowed")
	}

	sorts := make([]string, len(params))
	for i, param := range params {
		var sort string
		if param.Descending {
			sort = "desc"
		} else {
			sort = "asc"
		}

		sorts[i] = fmt.Sprintf("{%q: %q}", param.Field, sort)
	}

	if len(searchAfter) > 0 {
		searchAfterStringValues := make([]string, len(searchAfter))
		for i, v := range searchAfter {
			switch v.(type) {
			case string:
				searchAfterStringValues[i] = fmt.Sprintf("%q", v)
			default:
				searchAfterStringValues[i] = fmt.Sprintf("%v", v)
			}
		}

		return fmt.Sprintf(
			query,
			limit,
			strings.Join(sorts, ","),
			fmt.Sprintf(`
			,
			"search_after": [
				%s
			]`,
				strings.Join(searchAfterStringValues, ", ")),
		), nil
	}
	return fmt.Sprintf(query, limit, strings.Join(sorts, ","), ""), nil
}

func (paginator *ElasticPaginator) GetPlaces(limit int, offset int) ([]common.Place, int, error) {
	if offset < 0 {
		return nil, 0, fmt.Errorf("offset can not be less than 0")
	}
	places := make([]common.Place, 0)
	if limit == 0 {
		return places, 0, nil
	}

	var searchAfter []any = nil

	for i := 0; i < limit; i += 10_000 {
		query, err := buildQuery(
			10_000,
			searchAfter,
			[]SortParameter{
				{Field: "id", Descending: false},
				{Field: "_score", Descending: true},
			},
		)
		if err != nil {
			return nil, 0, err
		}

		response, err := paginator.Client.Search(
			paginator.Client.Search.WithIndex(paginator.Index),
			paginator.Client.Search.WithBody(
				strings.NewReader(query),
			),
		)
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

		// no more data to fetch
		if result.Hits.Hits == nil || len(result.Hits.Hits) == 0 {
			break
		}

		searchAfter = result.Hits.Hits[len(result.Hits.Hits)-1].Sort

		for _, place := range result.Hits.Hits {
			places = append(places, place.Source)
		}
	}

	if offset >= len(places) {
		return make([]common.Place, 0), 0, nil
	}

	if len(places) < offset+limit {
		places = places[offset:]
		return places, len(places), nil
	}

	places = places[offset : offset+limit]
	return places, len(places), nil
}
