package routes

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func UsersRouter() chi.Router {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "get all users"})
	})

	return r
}
