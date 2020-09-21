package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
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

func sendRecordsViaRFC2136(RRsToAdd []dns.RR, RRsToRemove []dns.RR, secret map[string]string, key string, domain string, algo string, server string) error {
	// initialization
	m := new(dns.Msg)
	m.SetUpdate(dns.Fqdn(domain))
	m.Insert(RRsToAdd)
	m.RemoveRRset(RRsToRemove)

	// Setup client
	c := &dns.Client{}
	c.SingleInflight = true

	// TSIG authentication / msg signing
	m.SetTsig(dns.Fqdn(key), dns.Fqdn(algo), 300, time.Now().Unix())
	c.TsigSecret = secret

	// Send the query
	reply, _, err := c.Exchange(m, server)
	if err != nil {
		return fmt.Errorf("DNS update failed: %w", err)
	}
	if reply != nil && reply.Rcode != dns.RcodeSuccess {
		return fmt.Errorf("DNS update failed: server replied: %s", dns.RcodeToString[reply.Rcode])
	}
	return nil
}

type GetAXFR struct {
	Keyname string `json:"keyName"`
	Key     string `json:"key"`
	Domain  string `json:"domain"`
	Algo    string `json:"algo"`
	Server  string `json:"server"`
}

type addNewRecords struct {
	KeyName    string `json:"keyname"`
	Domain     string `json:"domain"`
	Key        string `json:"key"`
	Algo       string `json:"algo"`
	Server     string `json:"server"`
	NewRecords []struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Type   string `json:"type"`
		Target string `json:"target"`
		TTL    string `json:"ttl"`
	} `json:"newRecords"`
	RemRecords []struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Type   string `json:"type"`
		Target string `json:"target"`
		TTL    string `json:"ttl"`
	} `json:"remRecords"`
}

func handlerSendRecords(w http.ResponseWriter, r *http.Request) {
	// Parses JSON
	vars := r.URL.Query()
	testtt := vars["data"]
	testttb := []byte(testtt[0])
	var request addNewRecords
	err := json.Unmarshal(testttb, &request)
	if err != nil {
		fmt.Println(err)
		fmt.Fprintf(w, "JSON is not valid.")
	}

	// actual "code"

	secret := map[string]string{request.KeyName + ".": request.Key}
	domain := request.Domain
	key := request.KeyName + "."
	algo := request.Algo + "." // my gosh that's important
	server := request.Server

	var records []dns.RR
	var toBeRemoved []dns.RR
	for _, currRecord := range request.NewRecords {
		textForNewRecord := currRecord.Name + " " + currRecord.TTL + " " + currRecord.Type + " " + currRecord.Target
		newRecord, _ := dns.NewRR(textForNewRecord)
		records = append(records, newRecord)
	}

	for _, currRecord := range request.RemRecords {
		textForNewRecord := currRecord.Name + " " + currRecord.TTL + " " + currRecord.Type + " " + currRecord.Target
		newRecord, _ := dns.NewRR(textForNewRecord)
		toBeRemoved = append(toBeRemoved, newRecord)

	}

	err = sendRecordsViaRFC2136(records, toBeRemoved, secret, key, domain, algo, server)

	w.Header().Set("Access-Control-Allow-Origin", "*")

	// return a JSON success
	fmt.Fprintf(w, `{"success":true}`)

	// poor man's log
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	var ip string
	if forwarded != "" {
		ip = forwarded
	} else {
		ip = r.RemoteAddr
	}
	log.WithFields(log.Fields{
		"Domain": domain,
		"IP":     ip,
	}).Info("GET /addRecords")
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

	secret := map[string]string{request.Keyname + ".": request.Key}
	domain := request.Domain
	key := request.Keyname + "."
	algo := request.Algo + "."
	server := request.Server
	records := getRecordsViaAXFR(secret, key, domain, algo, server)

	transfer := new(dns.Transfer)
	message := new(dns.Msg)
	transfer.TsigSecret = secret
	message.SetAxfr(domain)
	message.SetTsig(key, algo, 300, time.Now().Unix())

	jsonText, err := json.Marshal(records)
	if err != nil {
		fmt.Fprintln(w, err)
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprintf(w, string(jsonText))

	forwarded := r.Header.Get("X-FORWARDED-FOR")
	var ip string
	if forwarded != "" {
		ip = forwarded
	} else {
		ip = r.RemoteAddr
	}
	log.WithFields(log.Fields{
		"Domain": domain,
		"IP":     ip,
	}).Info("GET /getRecords")

}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hey it works!")
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	var ip string
	if forwarded != "" {
		ip = forwarded
	} else {
		ip = r.RemoteAddr
	}
	fmt.Println("GET / from " + ip)
	log.WithFields(log.Fields{
		"IP": ip,
	}).Info("GET /")
}

func main() {
	fmt.Println("Hello world!")
	r := mux.NewRouter()
	r.HandleFunc("/", mainHandler)
	r.HandleFunc("/getRecords", handlerGetAXFR).Methods("GET")
	r.HandleFunc("/addRecords", handlerSendRecords).Methods("GET")
	http.ListenAndServe(":8080", r)
}
