package api

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type API struct {
	DB *sql.DB
}

func New(db *sql.DB) *API {
	return &API{DB: db}
}

func (a *API) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/settings", func(r chi.Router) {
		r.Get("/", a.getSettings)
		r.Put("/", a.putSettings)
	})

	mountCRUD(r, "/machines", a.machines())
	mountCRUD(r, "/build-plates", a.buildPlates())
	mountCRUD(r, "/nozzles", a.nozzles())
	mountCRUD(r, "/ams-units", a.amsUnits())
	mountCRUD(r, "/other-equipment", a.otherEquipment())
	mountCRUD(r, "/filaments", a.filaments())
	mountCRUD(r, "/consumables", a.consumables())
	mountCRUD(r, "/material-settings", a.materialSettings())

	r.Route("/quotes", func(r chi.Router) {
		r.Get("/", a.listQuotes)
		r.Post("/", a.createQuote)
		r.Get("/{id}", a.getQuote)
		r.Put("/{id}", a.updateQuote)
		r.Delete("/{id}", a.deleteQuote)
		r.Post("/{id}/duplicate", a.duplicateQuote)
		r.Patch("/{id}/status", a.patchQuoteStatus)
	})

	r.Route("/reports", func(r chi.Router) {
		r.Get("/summary", a.reportSummary)
		r.Get("/revenue-over-time", a.reportRevenueOverTime)
		r.Get("/cost-breakdown-trends", a.reportCostBreakdownTrends)
		r.Get("/machine-utilisation", a.reportMachineUtilisation)
		r.Get("/filament-consumption", a.reportFilamentConsumption)
	})

	return r
}

// resource is the minimal interface a simple CRUD entity implements so it
// can be mounted generically by mountCRUD, avoiding repeating the five
// chi.Route calls for every one of the seven simple entity types.
type resource struct {
	List   http.HandlerFunc
	Create http.HandlerFunc
	Get    http.HandlerFunc
	Update http.HandlerFunc
	Delete http.HandlerFunc
}

func mountCRUD(r chi.Router, path string, res resource) {
	r.Route(path, func(r chi.Router) {
		r.Get("/", res.List)
		r.Post("/", res.Create)
		r.Get("/{id}", res.Get)
		r.Put("/{id}", res.Update)
		r.Delete("/{id}", res.Delete)
	})
}
