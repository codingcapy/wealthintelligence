package main

import (
	"fmt"
	"log"
	"net/http"

	"wealthintelligence/db"
	"wealthintelligence/routes"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

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
		r.Mount("/expenditures", routes.ExpendituresRouter())
		r.Mount("/assets", routes.AssetsRouter())
		r.Mount("/liabilities", routes.LiabilitiesRouter())
		r.Mount("/financialgoals", routes.FinancialGoalsRouter())
	})

	fmt.Println("Server running on port 3333")
	log.Fatal(http.ListenAndServe(":3333", r))
}
