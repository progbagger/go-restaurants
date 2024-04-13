package main

import (
	"args"
	"common"
	"db"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"paginate"
	"strconv"
	"strings"
)

const body = `
<!doctype html>
<html>
<head>
    <meta charset="utf-8">
    <title>Places</title>
    <meta name="description" content="">
    <meta name="viewport" content="width=device-width, initial-scale=1">
</head>

<body>
<h5>Total: %d</h5>
<h5>Current: %d</h5>
<ul>
%s
</ul>
%s
</body>
</html>`

func createButton(page int, name string) string {
	return fmt.Sprintf(`<a href="/?page=%d">%s</a>`, page, name)
}

func createPlaceEntry(place common.Place) string {
	return fmt.Sprintf(
		`
		<li>
			<div>%s</div>
			<div>%s</div>
			<div>%s</div>
		</li>`,
		place.Name,
		place.Address,
		place.Phone,
	)
}

func buildPage(total, pageSize, page int, places []common.Place) string {
	var stringPages []string
	if len(places) < pageSize {
		stringPages = make([]string, len(places))
	} else {
		stringPages = make([]string, pageSize)
	}

	for i, place := range places {
		stringPages[i] = createPlaceEntry(place)
	}

	entries := strings.Join(stringPages, "\n")

	var firstButton, prevButton, nextButton, lastButton string
	if page != 1 && total != 1 {
		firstButton = createButton(1, "First")
		prevButton = createButton(page-1, "Previous")
	}
	if page != total {
		nextButton = createButton(page+1, "Next")
		lastButton = createButton(total, "Last")
	}

	stringButtons := strings.Join(
		[]string{
			firstButton,
			prevButton,
			nextButton,
			lastButton,
		},
		"\n",
	)

	return fmt.Sprintf(body, total, page, entries, stringButtons)
}

const pageSize = 10

type Paginator struct {
	ElasticPaginator paginate.ElasticPaginator
}

func (paginator *Paginator) showPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "not a get method", http.StatusMethodNotAllowed)
		log.Println("not a get method")
		return
	}

	places, totalDocumentsCount, err := paginator.ElasticPaginator.GetPlaces(math.MaxInt32, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	requestedPage, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
		return
	}

	totalPagesCount := totalDocumentsCount / pageSize
	if totalDocumentsCount%10 != 0 {
		totalPagesCount++
	}

	if requestedPage <= 0 || requestedPage > int64(totalPagesCount) {
		http.Error(w, "requested page is invalid", http.StatusBadRequest)
		log.Println("requested page is invalid")
		return
	}

	sliceEnd := requestedPage * pageSize
	if len(places) < int(sliceEnd) {
		sliceEnd = int64(len(places))
	}

	fmt.Fprintln(w, buildPage(
		totalPagesCount,
		pageSize,
		int(requestedPage),
		places[(requestedPage-1)*pageSize:sliceEnd],
	))
}

type invalidPageJson struct {
	Error string `json:"error"`
}

type jsonResponse struct {
	Name      string         `json:"name"`
	Total     int            `json:"total"`
	Places    []common.Place `json:"places"`
	FirstPage *int           `json:"first_page,omitempty"`
	PrevPage  *int           `json:"prev_page,omitempty"`
	NextPage  *int           `json:"next_page,omitempty"`
	LastPage  *int           `json:"last_page,omitempty"`
}

func (paginator *Paginator) returnJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "not a get method", http.StatusMethodNotAllowed)
		log.Println("not a get method")
		return
	}

	places, totalDocumentsCount, err := paginator.ElasticPaginator.GetPlaces(math.MaxInt32, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	requestedPage, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		marshalized, _ := json.MarshalIndent(
			invalidPageJson{fmt.Sprintf("Invalid 'page' value: %v", r.URL.Query().Get("page"))},
			"",
			"  ",
		)
		http.Error(w, string(marshalized), http.StatusBadRequest)
		log.Println("requested page is invalid")
		return
	}

	totalPagesCount := totalDocumentsCount / pageSize
	if totalDocumentsCount%10 != 0 {
		totalPagesCount++
	}

	if requestedPage <= 0 || requestedPage > int64(totalPagesCount) {
		marshalized, _ := json.MarshalIndent(
			invalidPageJson{fmt.Sprintf("Invalid 'page' value: %v", requestedPage)},
			"",
			"  ",
		)
		http.Error(w, string(marshalized), http.StatusBadRequest)
		log.Println("requested page is invalid")
		return
	}

	sliceEnd := requestedPage * pageSize
	if len(places) < int(sliceEnd) {
		sliceEnd = int64(len(places))
	}
	places = places[(requestedPage-1)*pageSize : sliceEnd]

	response := jsonResponse{
		Name:   "Places",
		Total:  totalDocumentsCount,
		Places: places,
	}
	if requestedPage != 1 && totalPagesCount != 1 {
		response.FirstPage, response.PrevPage = new(int), new(int)
		*response.FirstPage, *response.PrevPage = 1, int(requestedPage)-1
	}
	if requestedPage != int64(totalPagesCount) {
		response.NextPage, response.LastPage = new(int), new(int)
		*response.NextPage, *response.LastPage = int(requestedPage)+1, totalPagesCount
	}

	marshalized, _ := json.MarshalIndent(response, "", "  ")
	fmt.Fprint(w, string(marshalized))
}

type recommendResponse struct {
	Name   string         `json:"name"`
	Places []common.Place `json:"places"`
}

type geoSortEntry struct {
	GeoDistance geoDistance `json:"_geo_distance"`
}

type geoDistance struct {
	Location       common.Location `json:"location"`
	Order          string          `json:"order"`
	Unit           string          `json:"unit"`
	Mode           string          `json:"mode"`
	DistanceType   string          `json:"distance_type"`
	IgnoreUnmapped bool            `json:"ignore_unmapped"`
}

type sortSizeRequest struct {
	Size int   `json:"size"`
	Sort []any `json:"sort"`
}

func constructGeoSortRequest(lon, lat float64, size int) sortSizeRequest {
	return sortSizeRequest{
		Size: size,
		Sort: []any{geoSortEntry{GeoDistance: geoDistance{
			Location: common.Location{
				Latitude:  lat,
				Longitude: lon,
			},
			Order:          "asc",
			Unit:           "km",
			Mode:           "min",
			DistanceType:   "arc",
			IgnoreUnmapped: true,
		}}}}
}

func (paginator *Paginator) recommendApi(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "not a get request", http.StatusMethodNotAllowed)
		log.Println("not a get request")
		return
	}

	lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	if err != nil {
		marshalized, _ := json.MarshalIndent(invalidPageJson{fmt.Sprintf(`invalid "lat" value: %v`, lat)}, "", "  ")
		http.Error(w, string(marshalized), http.StatusBadRequest)
		return
	}

	lon, err := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if err != nil {
		marshalized, _ := json.MarshalIndent(invalidPageJson{fmt.Sprintf(`invalid "lon" value: %v`, lon)}, "", "  ")
		http.Error(w, string(marshalized), http.StatusBadRequest)
		return
	}

	marshalizedSort, _ := json.MarshalIndent(constructGeoSortRequest(lon, lat, 3), "", "  ")
	log.Println(string(marshalizedSort))
	response, err := paginator.ElasticPaginator.Client.Search(
		paginator.ElasticPaginator.Client.Search.WithIndex(paginator.ElasticPaginator.Index),
		paginator.ElasticPaginator.Client.Search.WithBody(strings.NewReader(string(marshalizedSort))),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if response.IsError() {
		http.Error(w, response.String(), http.StatusInternalServerError)
	}

	var res paginate.ElasticSortResponse
	err = json.NewDecoder(response.Body).Decode(&res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	recommendResponse := recommendResponse{
		Name: "Recommend",
	}
	recommendResponse.Places = make([]common.Place, len(res.Hits.Hits))
	for i, hit := range res.Hits.Hits {
		recommendResponse.Places[i] = hit.Source
	}

	marshalizedResponse, _ := json.MarshalIndent(recommendResponse, "", "  ")
	fmt.Fprint(w, string(marshalizedResponse))
}

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

	paginator := Paginator{paginate.ElasticPaginator{Client: client, Index: "places"}}

	// handlers
	http.HandleFunc("/", paginator.showPage)
	http.HandleFunc("/api/places", paginator.returnJSON)
	http.HandleFunc("/api/recommend", paginator.recommendApi)

	// server itself
	err = http.ListenAndServe(":8888", nil)
	if err != nil {
		log.Fatalln(err)
	}
}