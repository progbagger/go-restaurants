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

func (paginator *ElasticPaginator) showPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusForbidden)
		log.Println("not a get method")
		return
	}

	places, totalDocumentsCount, err := paginator.GetPlaces(math.MaxInt32, 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	const pageSize = 10

	requestedPage, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return
	}

	totalPagesCount := totalDocumentsCount / pageSize
	if requestedPage <= 0 || requestedPage > int64(totalPagesCount) {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("requested page is invalid")
		return
	}

	fmt.Fprintln(w, buildPage(
		totalPagesCount,
		pageSize,
		int(requestedPage),
		places[(requestedPage-1)*pageSize:requestedPage*pageSize],
	))
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

	paginator := ElasticPaginator{Client: client, Index: "places"}

	// server itself
	http.HandleFunc("/", paginator.showPage)
	err = http.ListenAndServe(":8888", nil)
	if err != nil {
		log.Fatalln(err)
	}
}
