package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"

	"wealthintelligence/db"
	"wealthintelligence/models"

	"github.com/go-chi/chi/v5"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func AiRouter() chi.Router {
	r := chi.NewRouter()

	r.Post("/generate", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			PlanID int `json:"planId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Verify plan belongs to user
		var plan models.Plan
		err := db.DB.Get(&plan, "SELECT * FROM plans WHERE plan_id = $1 AND user = $2", body.PlanID, userID)
		if err != nil {
			http.Error(w, "Plan not found or unauthorized", http.StatusUnauthorized)
			return
		}

		// Fetch all related data
		var incomes []models.Income
		db.DB.Select(&incomes, "SELECT * FROM incomes WHERE plan_id = $1", body.PlanID)

		var expenditures []models.Expenditure
		db.DB.Select(&expenditures, "SELECT * FROM expenditures WHERE plan_id = $1", body.PlanID)

		var assets []models.Asset
		db.DB.Select(&assets, "SELECT * FROM assets WHERE plan_id = $1", body.PlanID)

		var liabilities []models.Liability
		db.DB.Select(&liabilities, "SELECT * FROM liabilities WHERE plan_id = $1", body.PlanID)

		var financialGoals []models.FinancialGoal
		db.DB.Select(&financialGoals, "SELECT * FROM financial_goals WHERE plan_id = $1", body.PlanID)

		// Calculate totals — amounts stored as cents
		totalIncome := 0.0
		for _, i := range incomes {
			totalIncome += (float64(i.Amount) / 100) * (100 - float64(i.Tax)/100) / 100
		}
		totalIncome = math.Round(totalIncome*100) / 100

		totalExpenditure := 0.0
		for _, e := range expenditures {
			totalExpenditure += float64(e.Amount) / 100
		}
		totalExpenditure = math.Round(totalExpenditure*100) / 100

		cashflow := totalIncome - totalExpenditure

		totalAssets := 0.0
		for _, a := range assets {
			totalAssets += float64(a.Value) / 100
		}
		totalAssets = math.Round(totalAssets*100) / 100

		totalLiabilities := 0.0
		for _, l := range liabilities {
			totalLiabilities += float64(l.Amount) / 100
		}
		totalLiabilities = math.Round(totalLiabilities*100) / 100

		netWorth := totalAssets - totalLiabilities

		// Build text summaries
		incomeParts := []string{}
		for _, i := range incomes {
			net := (float64(i.Amount) / 100) * (100 - float64(i.Tax)/100) / 100
			incomeParts = append(incomeParts, fmt.Sprintf("%s at %s: %s%.2f with tax %%%.2f therefore net %.2f",
				i.Position, i.Company, plan.Currency, float64(i.Amount)/100, float64(i.Tax)/100, net))
		}

		expenditureParts := []string{}
		for _, e := range expenditures {
			expenditureParts = append(expenditureParts, fmt.Sprintf("%s: %s%.2f", e.Name, plan.Currency, float64(e.Amount)/100))
		}

		assetParts := []string{}
		for _, a := range assets {
			assetParts = append(assetParts, fmt.Sprintf("%s: %s%.2f with ROI %%%.2f", a.Name, plan.Currency, float64(a.Value)/100, float64(a.ROI)/100))
		}

		liabilityParts := []string{}
		for _, l := range liabilities {
			liabilityParts = append(liabilityParts, fmt.Sprintf("%s: %s%.2f with interest %%%.2f", l.Name, plan.Currency, float64(l.Amount)/100, float64(l.Interest)/100))
		}

		goalParts := []string{}
		for _, g := range financialGoals {
			goalParts = append(goalParts, fmt.Sprintf("%s: %s%.2f with target date %s", g.Name, plan.Currency, float64(g.Amount)/100, g.TargetDate.Format("2006-01-02")))
		}

		prompt := fmt.Sprintf(`
Provide financial advice for the following client data

Year of birth: %s
Country of residence: %s
Currency: %s
Total income: %.2f
Total expenditure: %.2f
Cashflow: %.2f
Total assets: %.2f
Total liabilities: %.2f
Net worth: %.2f

INCOME SOURCES (monthly):
%s

EXPENDITURES (monthly):
%s

ASSETS:
%s

LIABILITIES:
%s

FINANCIAL GOALS:
%s

Limit your advice to no more than 1500 characters
ALWAYS take into consideration the client's age and country of residence and whether this means they are willing to take more risks or retired and comfortable or trying to raise a family
ALWAYS start with "Based on the provided financial data, here's a comprehensive analysis of your"`,
			plan.YearOfBirth, plan.Location, plan.Currency,
			totalIncome, totalExpenditure, cashflow,
			totalAssets, totalLiabilities, netWorth,
			strings.Join(incomeParts, "\n"),
			strings.Join(expenditureParts, "\n"),
			strings.Join(assetParts, "\n"),
			strings.Join(liabilityParts, "\n"),
			strings.Join(goalParts, "\n"),
		)

		// Call OpenAI
		client := openai.NewClient(option.WithAPIKey(os.Getenv("OPENAI_API_KEY")))
		completion, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
			Model:       openai.ChatModelGPT4o,
			Temperature: openai.Float(0.4),
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage("You are a certified financial planner providing clear financial advice."),
				openai.UserMessage(prompt),
			},
		})
		if err != nil {
			http.Error(w, "Error generating recommendation", http.StatusInternalServerError)
			return
		}

		content := completion.Choices[0].Message.Content

		// Save generation to DB
		var generation models.Generation
		err = db.DB.QueryRowx(
			`INSERT INTO generations (plan_id, content) VALUES ($1, $2) RETURNING *`,
			body.PlanID, content,
		).StructScan(&generation)
		if err != nil {
			http.Error(w, "Error saving generation", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"generation": generation})
	})

	return r
}
