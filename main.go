package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	log "github.com/sirupsen/logrus"

	"github.com/rs/cors"
)

type Todo struct {
	Task string `json:"task"`
	ID   string `json:"id"`
}

var db *sql.DB

func initDB() {
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "password")
	dbname := getEnv("POSTGRES_DB", "todos")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Create table if not exists
	createTable := `
	CREATE TABLE IF NOT EXISTS todos (
		id VARCHAR(36) PRIMARY KEY,
		task TEXT NOT NULL
	);`

	if _, err = db.Exec(createTable); err != nil {
		log.Fatal("Failed to create table:", err)
	}

	log.Info("Database connection established")
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func createItem(w http.ResponseWriter, r *http.Request) {
	task := r.FormValue("task")
	todo := Todo{Task: task, ID: uuid.New().String()}
	
	_, err := db.Exec("INSERT INTO todos (id, task) VALUES ($1, $2)", todo.ID, todo.Task)
	if err != nil {
		log.Error("Failed to save todo item:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	
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

	result, err := db.Exec("DELETE FROM todos WHERE id = $1", id)
	if err != nil {
		log.Error("Failed to delete todo item:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("Failed to get rows affected:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Info("deleted todo item")
}

func getItems(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, task FROM todos ORDER BY task DESC")
	if err != nil {
		log.Error("Failed to query todos:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var all []Todo
	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.Task)
		if err != nil {
			log.Error("Failed to scan todo row:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		all = append(all, todo)
	}

	if err = rows.Err(); err != nil {
		log.Error("Rows iteration error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(all)
	log.Infof("got %d items", len(all))
}

func main() {
	initDB()
	defer db.Close()

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
