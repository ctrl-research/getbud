package server

import (
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/ctrl-research/getbud/internal/auth"
	"github.com/ctrl-research/getbud/internal/store"
	"github.com/ctrl-research/getbud/internal/store/sqlcgen"
)

// reportRange reads from/to query params, defaulting to the last 12 months.
func reportRange(w http.ResponseWriter, r *http.Request) (from, to time.Time, ok bool) {
	now := time.Now()
	to = now
	from = now.AddDate(-1, 0, 0)
	if v := r.URL.Query().Get("from"); v != "" {
		d, valid := parseDate(v)
		if !valid {
			apiError(w, http.StatusBadRequest, "invalid_date", "from must be YYYY-MM-DD")
			return from, to, false
		}
		from = d
	}
	if v := r.URL.Query().Get("to"); v != "" {
		d, valid := parseDate(v)
		if !valid {
			apiError(w, http.StatusBadRequest, "invalid_date", "to must be YYYY-MM-DD")
			return from, to, false
		}
		to = d
	}
	if to.Before(from) {
		apiError(w, http.StatusBadRequest, "invalid_range", "to must be on or after from")
		return from, to, false
	}
	return from, to, true
}

type categoryFlowJSON struct {
	CategoryID  *uuid.UUID `json:"categoryId"`
	Name        string     `json:"name"`
	Kind        string     `json:"kind"`
	Color       string     `json:"color"`
	AmountCents int64      `json:"amountCents"`
}

// splitCategoryTotals turns raw per-category flows into income and expense
// lists. Categories follow their kind (net of refunds); uncategorized rows
// bucket by sign.
func splitCategoryTotals(rows []store.CategoryTotalsRow) (income, expense []categoryFlowJSON) {
	for _, row := range rows {
		switch {
		case row.CategoryKind != nil && *row.CategoryKind == sqlcgen.CategoryKindIncome:
			if net := row.InflowCents - row.OutflowCents; net > 0 {
				income = append(income, categoryFlowJSON{CategoryID: row.CategoryID, Name: row.CategoryName, Kind: "income", Color: row.CategoryColor, AmountCents: net})
			}
		case row.CategoryKind != nil && *row.CategoryKind == sqlcgen.CategoryKindExpense:
			if net := row.OutflowCents - row.InflowCents; net > 0 {
				expense = append(expense, categoryFlowJSON{CategoryID: row.CategoryID, Name: row.CategoryName, Kind: "expense", Color: row.CategoryColor, AmountCents: net})
			}
		default: // uncategorized: bucket by sign, both sides possible
			if row.InflowCents > 0 {
				income = append(income, categoryFlowJSON{Name: "Uncategorized", Kind: "income", AmountCents: row.InflowCents})
			}
			if row.OutflowCents > 0 {
				expense = append(expense, categoryFlowJSON{Name: "Uncategorized", Kind: "expense", AmountCents: row.OutflowCents})
			}
		}
	}
	sort.Slice(income, func(i, j int) bool { return income[i].AmountCents > income[j].AmountCents })
	sort.Slice(expense, func(i, j int) bool { return expense[i].AmountCents > expense[j].AmountCents })
	return income, expense
}

func (api *budgetAPI) reportSummary(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	from, to, ok := reportRange(w, r)
	if !ok {
		return
	}
	summary, err := api.reports.Summary(r.Context(), user.ID, from, to)
	if err != nil {
		apiInternalError(w, "report summary", err)
		return
	}
	totals, err := api.reports.CategoryTotals(r.Context(), user.ID, from, to)
	if err != nil {
		apiInternalError(w, "report category totals", err)
		return
	}
	_, expense := splitCategoryTotals(totals)
	if len(expense) > 5 {
		expense = expense[:5]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from":                 from.Format(dateOnly),
		"to":                   to.Format(dateOnly),
		"incomeCents":          summary.IncomeCents,
		"expenseCents":         summary.ExpenseCents,
		"netCents":             summary.IncomeCents - summary.ExpenseCents,
		"uncategorizedCount":   summary.UncategorizedCount,
		"topExpenseCategories": expense,
	})
}

// reportSankey builds income category -> Income -> expense category flows.
// The expense side keeps the top N categories and folds the rest into
// "Other" so the diagram stays readable; unspent income flows to "Savings".
func (api *budgetAPI) reportSankey(w http.ResponseWriter, r *http.Request) {
	const maxExpenseNodes = 12
	user, _ := auth.UserFrom(r.Context())
	from, to, ok := reportRange(w, r)
	if !ok {
		return
	}
	totals, err := api.reports.CategoryTotals(r.Context(), user.ID, from, to)
	if err != nil {
		apiInternalError(w, "report category totals", err)
		return
	}
	income, expense := splitCategoryTotals(totals)

	if len(expense) > maxExpenseNodes {
		var otherCents int64
		for _, e := range expense[maxExpenseNodes:] {
			otherCents += e.AmountCents
		}
		expense = append(expense[:maxExpenseNodes], categoryFlowJSON{Name: "Other", Kind: "expense", AmountCents: otherCents})
	}

	type node struct {
		Name  string `json:"name"`
		Color string `json:"color,omitempty"`
	}
	type link struct {
		Source     string `json:"source"`
		Target     string `json:"target"`
		ValueCents int64  `json:"valueCents"`
	}
	var nodes []node
	var links []link

	var totalIncome, totalExpense int64
	for _, in := range income {
		name := in.Name
		if name == "Uncategorized" {
			name = "Uncategorized income"
		}
		nodes = append(nodes, node{Name: name, Color: in.Color})
		links = append(links, link{Source: name, Target: "Income", ValueCents: in.AmountCents})
		totalIncome += in.AmountCents
	}
	nodes = append(nodes, node{Name: "Income"})
	for _, ex := range expense {
		name := ex.Name
		if name == "Uncategorized" {
			name = "Uncategorized spending"
		}
		nodes = append(nodes, node{Name: name, Color: ex.Color})
		links = append(links, link{Source: "Income", Target: name, ValueCents: ex.AmountCents})
		totalExpense += ex.AmountCents
	}
	// The remainder keeps the Sankey balanced: spent more than earned shows
	// as a "From savings" inflow, less as a "Savings" outflow.
	switch {
	case totalIncome > totalExpense:
		nodes = append(nodes, node{Name: "Savings"})
		links = append(links, link{Source: "Income", Target: "Savings", ValueCents: totalIncome - totalExpense})
	case totalExpense > totalIncome:
		nodes = append(nodes, node{Name: "From savings"})
		links = append(links, link{Source: "From savings", Target: "Income", ValueCents: totalExpense - totalIncome})
	}

	if nodes == nil {
		nodes = []node{}
	}
	if links == nil {
		links = []link{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from": from.Format(dateOnly), "to": to.Format(dateOnly),
		"nodes": nodes, "links": links,
	})
}

// reportTrends returns the month x category matrix for trend charts.
func (api *budgetAPI) reportTrends(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	from, to, ok := reportRange(w, r)
	if !ok {
		return
	}
	rows, err := api.reports.Trends(r.Context(), user.ID, from, to)
	if err != nil {
		apiInternalError(w, "report trends", err)
		return
	}
	type trendJSON struct {
		Month        string     `json:"month"`
		CategoryID   *uuid.UUID `json:"categoryId"`
		Name         string     `json:"name"`
		Kind         string     `json:"kind"`
		Color        string     `json:"color"`
		InflowCents  int64      `json:"inflowCents"`
		OutflowCents int64      `json:"outflowCents"`
	}
	out := make([]trendJSON, 0, len(rows))
	for _, row := range rows {
		kind := ""
		if row.CategoryKind != nil {
			kind = string(*row.CategoryKind)
		}
		out = append(out, trendJSON{
			Month: row.Month.Format(dateOnly), CategoryID: row.CategoryID,
			Name: row.CategoryName, Kind: kind, Color: row.CategoryColor,
			InflowCents: row.InflowCents, OutflowCents: row.OutflowCents,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from": from.Format(dateOnly), "to": to.Format(dateOnly),
		"rows": out,
	})
}

// reportNetWorth returns the monthly carry-forward net worth per account type.
func (api *budgetAPI) reportNetWorth(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	from, to, ok := reportRange(w, r)
	if !ok {
		return
	}
	rows, err := api.reports.NetWorth(r.Context(), user.ID, from, to)
	if err != nil {
		apiInternalError(w, "report net worth", err)
		return
	}
	type netWorthJSON struct {
		Month      string `json:"month"`
		Type       string `json:"type"`
		TotalCents int64  `json:"totalCents"`
	}
	out := make([]netWorthJSON, 0, len(rows))
	for _, row := range rows {
		out = append(out, netWorthJSON{Month: row.Month.Format(dateOnly), Type: string(row.AccountType), TotalCents: row.TotalCents})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from": from.Format(dateOnly), "to": to.Format(dateOnly),
		"rows": out,
	})
}
