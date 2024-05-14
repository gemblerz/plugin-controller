package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type APIServer struct {
	version    string
	port       int
	mainRouter *mux.Router
}

func NewAPIServer() *APIServer {
	return &APIServer{
		port: 9100,
	}
}

func (api *APIServer) Run(prometheusGatherer *prometheus.Registry) {
	api_address_port := fmt.Sprintf("0.0.0.0:%d", api.port)
	log.Printf("API server starts at %q...", api_address_port)
	api.mainRouter = mux.NewRouter()
	r := api.mainRouter
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"id": "Plugin Controller", "version":"`+api.version+`"}`)
	})

	if prometheusGatherer != nil {
		r.Handle("/metrics",
			promhttp.HandlerFor(prometheusGatherer, promhttp.HandlerOpts{EnableOpenMetrics: true})).
			Methods(http.MethodGet)
	}
	// api_route := r.PathPrefix("/api/v1").Subrouter()
	// api_route.Handle("/kb/rules", http.HandlerFunc(api.handlerRules)).Methods(http.MethodGet, http.MethodPost)
	log.Fatalln(http.ListenAndServe(api_address_port, handlers.LoggingHandler(os.Stdout, r)))
}

func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// json.NewEncoder(w).Encode(data)
	s, err := json.MarshalIndent(data, "", "  ")
	if err == nil {
		w.Write(s)
	}
}
