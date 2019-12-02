package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"github.com/oklog/ulid"
)

type newReq struct {
	Resolution float64
	MaxTime    int
	Keys       []string
}
type addReq struct {
	ID    string
	Kvs   map[string]string
	Time  int
	Value float64
	Count int
}

type deleteReq struct {
	ID string
}

type quantileReq struct {
	ID     string
	Kvs    map[string]string
	Time   int
	Quants []float64
}

var (
	entropy  = rand.New(rand.NewSource(int64(ulid.Now())))
	registry = make(map[string]*HMST)
)

// ServeNew makes a new sketch and store it, returning the ID of the sketch
func ServeNew(w http.ResponseWriter, r *http.Request) {
	var data newReq

	ID, err := ulid.New(ulid.Now(), entropy)
	if err != nil {
		log.Println(err)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Println(err)
		return
	}
	newH := NewHMST(data.Resolution, data.MaxTime, data.Keys)
	registry[ID.String()] = newH
	log.Println("/new", ID.String(), data, len(newH.Registers))
	fmt.Fprintf(w, "%v", ID)
}

// ServeAdd adds an item to a sketch
func ServeAdd(w http.ResponseWriter, r *http.Request) {
	var data addReq
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Println(err)
		return
	}
	temp, ok := registry[data.ID]
	if !ok {
		log.Println("/add", data.ID, "not found")
		fmt.Fprintf(w, "ID not found")
		return
	}
	(*temp).Add(data.Kvs, data.Time, data.Value, data.Count)
	log.Println("/add", data)
	rec := recover()
	if rec != nil {
		fmt.Fprintf(w, "%v", rec)
	} else {
		fmt.Fprintf(w, "ok")
	}
}

// ServeQuantiles returns quantiles as specified for the held values of the specified location and ID
func ServeQuantiles(w http.ResponseWriter, r *http.Request) {
	var data quantileReq
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Println(err)
	}
	val, ok := registry[data.ID]
	if !ok {
		log.Println("/quantiles ", data.ID, " not found")
		fmt.Fprintf(w, "ID not found")
	} else {
		sketch := (*val).Sketch(data.Kvs, data.Time)
		qs := Quantile(sketch, data.Quants)
		log.Println("/quantiles", data.ID, qs)
		fmt.Fprintf(w, "%v", qs)
	}

}

// ServeDelete deletes objects from the registry. Designed with an obvious
// lack of concern for security and abuse, because wtf?
func ServeDelete(w http.ResponseWriter, r *http.Request) {
	var data deleteReq
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Println(err)
	}
	if _, ok := registry[data.ID]; ok {
		delete(registry, data.ID)
		fmt.Fprintf(w, "ok")
		log.Println("/delete", data.ID, "success")
	} else {
		fmt.Fprintf(w, "ID not found")
		log.Println("/delete", data.ID, "failure")
	}

}

// ServeAPI returns a description of what the server can do
func ServeAPI(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `<!doctype html><body>
Refer <a href='https://gitlab.com/c25l/hmst/blob/master/server_test.go'>here</a> for some querying examples and details.
</body>`)
	log.Println("/ request")
}

func start_server() {
	http.HandleFunc("/new", ServeNew)
	http.HandleFunc("/add", ServeAdd)
	http.HandleFunc("/quantiles", ServeQuantiles)
	http.HandleFunc("/delete", ServeDelete)
	http.HandleFunc("/", ServeAPI)

	log.Fatal(http.ListenAndServe(":30903", nil))
}
