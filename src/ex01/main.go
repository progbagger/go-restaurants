package main

import (
	"args"
	"common"
	"db"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
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
<ul>
%s
</ul>
%s
</body>
</html>`

func (paginator *ElasticPaginator) buildPage(total int, places []common.Place) (string, error) {
	// ...
}

func (paginator *ElasticPaginator) showPage(w http.ResponseWriter, r *http.Request) {
	_, total, err := paginator.GetPlaces(math.MaxInt32, 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	const pageSize = 10

	requestedPage, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	b := strings.Builder{}
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

	// server itself
	err = http.ListenAndServe(":8888", nil)
	if err != nil {
		log.Fatalln(err)
	}

	var limit, offset int
	fmt.Scan(&limit, &offset)

	paginator := ElasticPaginator{Client: client, Index: "places"}
	places, hits, err := paginator.GetPlaces(limit, offset)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("hits: %d\n", hits)
	fmt.Printf("len: %d\n", len(places))

	for _, place := range places {
		fmt.Println(place)
	}
}
