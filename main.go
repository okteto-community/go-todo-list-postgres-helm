package main

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"

	"github.com/rs/cors"
)

type Todo struct {
	Task string `json:"task"`
	ID   string `json:"id"`
}

var store = map[string]Todo{}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func createItem(w http.ResponseWriter, r *http.Request) {
	task := r.FormValue("task")
	todo := Todo{Task: task, ID: uuid.New().String()}
	store[todo.ID] = todo
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
	log.Info("saved todo item")
}

func deleteItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if _, ok := store[id]; !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	delete(store, id)

	w.WriteHeader(http.StatusOK)
	log.Info("deleted todo item")
}

func getItems(w http.ResponseWriter, r *http.Request) {

	if len(store) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Todo{})
		return
	}

	var all []Todo
	for _, todo := range store {
		all = append(all, todo)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Task > all[j].Task
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(all)
	log.Infof("got %d items", len(all))
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/healthz", healthz).Methods("GET")
	router.HandleFunc("/todo", getItems).Methods("GET")
	router.HandleFunc("/todo", createItem).Methods("POST")
	router.HandleFunc("/todo/{id}", deleteItem).Methods("DELETE")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	handler := cors.New(cors.Options{
		AllowedMethods: []string{"GET", "POST", "DELETE", "PATCH", "OPTIONS"},
	}).Handler(router)

	log.Info("Starting API server...")
	http.ListenAndServe(":8080", handler)
}
