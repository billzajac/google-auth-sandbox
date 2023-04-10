package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", HelloServer)
	http.ListenAndServe(":9999", nil)
}

func HelloServer(w http.ResponseWriter, r *http.Request) {
	write_n_log(w, fmt.Sprintf("Hello, %s!", r.URL.Path[1:]))
}

func write_n_log(w http.ResponseWriter, s string) {
	fmt.Fprintf(w, "%s", s)
	log.Printf("%s", s)
}
