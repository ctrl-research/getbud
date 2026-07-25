package server

import (
	"net/http"

	"github.com/ctrl-research/getbud/internal/auth"
)

func (api *budgetAPI) routes(mux *http.ServeMux) {
	protect := func(pattern string, h http.HandlerFunc) {
		mux.Handle(pattern, auth.RequireUser(h))
	}

	protect("GET /api/v1/accounts", api.listAccounts)
	protect("POST /api/v1/accounts", api.createAccount)
	protect("PATCH /api/v1/accounts/{id}", api.updateAccount)
	protect("DELETE /api/v1/accounts/{id}", api.deleteAccount)
	protect("GET /api/v1/accounts/{id}/snapshots", api.listSnapshots)
	protect("PUT /api/v1/accounts/{id}/snapshots", api.upsertSnapshot)
	protect("DELETE /api/v1/accounts/{id}/snapshots/{snapshotId}", api.deleteSnapshot)

	protect("GET /api/v1/categories", api.listCategories)
	protect("POST /api/v1/categories", api.createCategory)
	protect("PATCH /api/v1/categories/{id}", api.updateCategory)
	protect("DELETE /api/v1/categories/{id}", api.deleteCategory)

	protect("GET /api/v1/transactions", api.listTransactions)
	protect("POST /api/v1/transactions", api.createTransaction)
	protect("POST /api/v1/transactions/transfer", api.createTransfer)
	protect("PATCH /api/v1/transactions/{id}", api.updateTransaction)
	protect("DELETE /api/v1/transactions/{id}", api.deleteTransaction)

	protect("GET /api/v1/contribution-room", api.getContributionRoom)
	protect("PUT /api/v1/contribution-room/{type}/{year}", api.putContributionRoom)

	protect("POST /api/v1/imports/preview", api.previewImport)
	protect("POST /api/v1/imports", api.commitImport)
	protect("GET /api/v1/imports", api.listImports)
	protect("DELETE /api/v1/imports/{id}", api.revertImport)

	protect("GET /api/v1/reports/summary", api.reportSummary)
	protect("GET /api/v1/reports/sankey", api.reportSankey)
	protect("GET /api/v1/reports/trends", api.reportTrends)
	protect("GET /api/v1/reports/net-worth", api.reportNetWorth)
}
