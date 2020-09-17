package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/luclu7/dns"
	"net/http"
	"time"
)

func getRecordsViaAXFR(secret map[string]string, key string, domain string, algo string, server string) []dns.RR {
	t := new(dns.Transfer)
	m := new(dns.Msg)
	t.TsigSecret = secret
	m.SetAxfr(domain)
	m.SetTsig(key, algo, 300, time.Now().Unix())
	c, err := t.In(m, server)
	if err != nil {
		panic(err)
	}
	var a []dns.RR
	for r := range c {
		a = r.RR
	}
	return a
}

type GetAXFR struct {
	KeyName string `json:"keyName"`
	Key     string `json:"key"`
	Domain  string `json:"domain"`
	Algo    string `json:"algo"`
	Server  string `json:"server"`
}

func handlerSendAXFR(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func handlerGetAXFR(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	testtt := vars["data"]
	testttb := []byte(testtt[0])
	var request GetAXFR
	err := json.Unmarshal(testttb, &request)
	if err != nil {
		fmt.Println(err)
		fmt.Fprintf(w, "JSON is not valid.")
	}

	secret := map[string]string{request.KeyName + ".": request.Key}
	domain := request.Domain
	key := request.KeyName + "."
	algo := request.Algo + "."
	server := request.Server
	fmt.Println("secret: ", secret, " ", key, " ", domain, " ", server)
	records := getRecordsViaAXFR(secret, key, domain, algo, server)

	mx, err := dns.NewRR("miek.nl MX 10 mx.miek.nl")
	records = append(records, mx)

	transfer := new(dns.Transfer)
	message := new(dns.Msg)
	transfer.TsigSecret = secret
	message.SetAxfr(domain)
	message.SetTsig(key, algo, 300, time.Now().Unix())
	c, err := transfer.In(message, server)
	if err != nil {
		panic(err)
	}
	//var a []dns.RR
	for r := range c {
		//a = r.RR
		fmt.Println(r.RR)
	}

	jsonText, err := json.Marshal(records)
	if err != nil {
		fmt.Fprintln(w, err)
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprintf(w, string(jsonText))

}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "lalala")
	fmt.Println("lalala")
}

func main() {
	fmt.Println("Hello world!")
	r := mux.NewRouter()
	r.HandleFunc("/", mainHandler)
	r.HandleFunc("/getRecords", handlerGetAXFR).Methods("GET")
	//r.HandleFunc("/getRecords", mainHandler).Methods("GET")
	http.ListenAndServe(":8080", r)
}
