package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type APIServer struct {
	version    string
	port       int
	mainRouter *mux.Router
}

func (api *APIServer) Run() {
	api_address_port := fmt.Sprintf("0.0.0.0:%d", api.port)
	log.Printf("API server starts at %q...", api_address_port)
	api.mainRouter = mux.NewRouter()
	r := api.mainRouter
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"id": "Plugin Controller", "version":"`+api.version+`"}`)
	})
	// api_route := r.PathPrefix("/api/v1").Subrouter()
	// api_route.Handle("/kb/rules", http.HandlerFunc(api.handlerRules)).Methods(http.MethodGet, http.MethodPost)
	log.Fatalln(http.ListenAndServe(api_address_port, r))
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
