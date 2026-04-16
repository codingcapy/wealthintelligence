package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"wealthintelligence/db"
	"wealthintelligence/routes"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

//go:embed frontend/dist
var staticFiles embed.FS

func main() {
	godotenv.Load()
	db.Connect()

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Route("/api/v0", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})
		r.Mount("/users", routes.UsersRouter())
		r.Mount("/plans", routes.PlansRouter())
		r.Mount("/incomes", routes.IncomesRouter())
		r.Mount("/liabilities", routes.LiabilitiesRouter())
		r.Mount("/assets", routes.AssetsRouter())
		r.Mount("/expenditures", routes.ExpendituresRouter())
		r.Mount("/financialgoals", routes.FinancialGoalsRouter())
		r.Mount("/generations", routes.GenerationsRouter())
		r.Mount("/user", routes.UserRouter())
		r.Mount("/ai", routes.AiRouter())
	})

	stripped, _ := fs.Sub(staticFiles, "frontend/dist")
	fileServer := http.FileServer(http.FS(stripped))

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		_, err := stripped.Open(path)
		if err != nil {
			r.URL.Path = "/"
			path = "index.html"
		}

		switch {
		case strings.HasSuffix(path, ".js"):
			w.Header().Set("Content-Type", "application/javascript")
		case strings.HasSuffix(path, ".css"):
			w.Header().Set("Content-Type", "text/css")
		case strings.HasSuffix(path, ".svg"):
			w.Header().Set("Content-Type", "image/svg+xml")
		case strings.HasSuffix(path, ".png"):
			w.Header().Set("Content-Type", "image/png")
		case strings.HasSuffix(path, ".ico"):
			w.Header().Set("Content-Type", "image/x-icon")
		}

		fileServer.ServeHTTP(w, r)
	})

	fmt.Println("Server running on port 3333")
	log.Fatal(http.ListenAndServe(":3333", r))
}
