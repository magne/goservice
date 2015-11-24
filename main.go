package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/enbritely/heartbeat-golang"
)

var queries = []Query{}

const (
	EnvHeartbeatAddress = "HEARTBEAT_ADDRESS"
	EnvListeningAddress = "LISTENING_ADDRESS"
)

type M struct {
	handler http.Handler
}

func (m M) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	start := time.Now()
	m.handler.ServeHTTP(rw, r)
	log.Printf("%s served in %s\n", r.URL, time.Since(start))
}

func NewM(h http.Handler) http.Handler {
	return M{h}
}

type IPMessage struct {
	IPs []net.IP
}

type ErrorMessage struct {
	Error string
}

func ipHandler(rw http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if len(domain) == 0 {
		json.NewEncoder(rw).Encode(ErrorMessage{"Empty domain parameter"})
		return
	}
	ips, err := net.LookupIP(domain)
	if err != nil {
		json.NewEncoder(rw).Encode(ErrorMessage{"Invalid domain address."})
		return
	}
	json.NewEncoder(rw).Encode(IPMessage{ips})
	queries = append(queries, Query{domain, ips})
}

type Query struct {
	Domain string
	IPs    []net.IP
}

type PageData struct {
	Build   string
	Queries []Query
}

func uiHandler(rw http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Panic("Error occured parsing the template", err)
	}
	page := PageData{
		Build:   heartbeat.CommitHash,
		Queries: queries,
	}
	if err = tmpl.Execute(rw, page); err != nil {
		log.Panic("Failed to write template", err)
	}

}

func createBaseHandler() http.Handler {
	r := http.NewServeMux()
	r.HandleFunc("/service/ip", ipHandler)
	r.HandleFunc("/", uiHandler)
	return NewM(r)
}

func main() {
	log.SetPrefix("[service] ")

	hAddress := os.Getenv(EnvHeartbeatAddress)
	go heartbeat.RunHeartbeatService(hAddress)

	address := os.Getenv(EnvListeningAddress)
	log.Println("IP service request at: " + address)
	log.Println(http.ListenAndServe(address, createBaseHandler()))
}
