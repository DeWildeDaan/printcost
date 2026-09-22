package api

import (
	"net/http"
	"sort"
)

// --- summary ---

type summaryResponse struct {
	TotalQuotes      int     `json:"total_quotes"`
	TotalRevenue     float64 `json:"total_revenue"`
	AverageMarginPct float64 `json:"average_margin_pct"`
	MostUsedFilament string  `json:"most_used_filament"`
}

func (a *API) reportSummary(w http.ResponseWriter, r *http.Request) {
	resp := summaryResponse{}

	if err := a.DB.QueryRow(`SELECT COUNT(*) FROM quotes`).Scan(&resp.TotalQuotes); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := a.DB.QueryRow(
		`SELECT COALESCE(SUM(suggested_price), 0) FROM quotes WHERE status = 'paid'`,
	).Scan(&resp.TotalRevenue); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var avgMargin sql64
	if err := a.DB.QueryRow(
		`SELECT AVG(CASE WHEN cost_total > 0 THEN cost_markup / cost_total * 100 ELSE 0 END)
		 FROM quotes WHERE status = 'paid'`,
	).Scan(&avgMargin); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp.AverageMarginPct = avgMargin.value

	usage, err := a.filamentUsage()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	bestGrams := -1.0
	for _, u := range usage {
		if u.Grams > bestGrams {
			bestGrams = u.Grams
			resp.MostUsedFilament = u.Name
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// sql64 scans a nullable REAL/AVG result (NULL when there are no matching rows) as 0.
type sql64 struct{ value float64 }

func (s *sql64) Scan(src any) error {
	if src == nil {
		s.value = 0
		return nil
	}
	switch v := src.(type) {
	case float64:
		s.value = v
	case int64:
		s.value = float64(v)
	}
	return nil
}

// --- revenue & quote volume over time (last 12 months) ---

type monthPoint struct {
	Month string  `json:"month"`
	Value float64 `json:"value"`
}

func (a *API) reportRevenueOverTime(w http.ResponseWriter, r *http.Request) {
	revenue, err := a.monthlySeries(
		`SELECT strftime('%Y-%m', created_at) AS m, COALESCE(SUM(suggested_price), 0)
		 FROM quotes WHERE status = 'paid' AND created_at >= datetime('now', '-12 months')
		 GROUP BY m`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	counts, err := a.monthlySeries(
		`SELECT strftime('%Y-%m', created_at) AS m, COUNT(*)
		 FROM quotes WHERE status != 'cancelled' AND created_at >= datetime('now', '-12 months')
		 GROUP BY m`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"revenue_by_month": revenue,
		"quotes_by_month":  counts,
	})
}

func (a *API) monthlySeries(query string) ([]monthPoint, error) {
	rows, err := a.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	points := []monthPoint{}
	for rows.Next() {
		var p monthPoint
		if err := rows.Scan(&p.Month, &p.Value); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Month < points[j].Month })
	return points, nil
}

// --- cost breakdown trends (last 12 months, % of total per month) ---

type costBreakdownMonth struct {
	Month           string  `json:"month"`
	FilamentPct     float64 `json:"filament_pct"`
	ElectricityPct  float64 `json:"electricity_pct"`
	DepreciationPct float64 `json:"depreciation_pct"`
	LaborPct        float64 `json:"labor_pct"`
	ConsumablesPct  float64 `json:"consumables_pct"`
	FailurePct      float64 `json:"failure_pct"`
	MarkupPct       float64 `json:"markup_pct"`
}

func (a *API) reportCostBreakdownTrends(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(
		`SELECT strftime('%Y-%m', created_at) AS m,
			SUM(cost_filament), SUM(cost_electricity), SUM(cost_depreciation),
			SUM(cost_labor), SUM(cost_consumables), SUM(cost_failure), SUM(cost_markup)
		 FROM quotes
		 WHERE status != 'cancelled' AND created_at >= datetime('now', '-12 months')
		 GROUP BY m`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := []costBreakdownMonth{}
	for rows.Next() {
		var m costBreakdownMonth
		var filament, electricity, depreciation, labor, consumables, failure, markup float64
		if err := rows.Scan(&m.Month, &filament, &electricity, &depreciation, &labor, &consumables, &failure, &markup); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		total := filament + electricity + depreciation + labor + consumables + failure + markup
		if total > 0 {
			m.FilamentPct = filament / total * 100
			m.ElectricityPct = electricity / total * 100
			m.DepreciationPct = depreciation / total * 100
			m.LaborPct = labor / total * 100
			m.ConsumablesPct = consumables / total * 100
			m.FailurePct = failure / total * 100
			m.MarkupPct = markup / total * 100
		}
		result = append(result, m)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Month < result[j].Month })
	writeJSON(w, http.StatusOK, result)
}

// --- machine utilisation ---

type machineUtilisation struct {
	Machine           string  `json:"machine"`
	TotalHours        float64 `json:"total_hours"`
	TotalDepreciation float64 `json:"total_depreciation"`
}

func (a *API) reportMachineUtilisation(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(
		`SELECT m.name,
			COALESCE(SUM((q.print_time_min + q.prototype_time_min) / 60.0), 0),
			COALESCE(SUM(q.cost_depreciation), 0)
		 FROM machines m
		 LEFT JOIN quotes q ON q.machine_id = m.id AND q.status != 'cancelled'
		 GROUP BY m.id
		 ORDER BY 2 DESC`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	list := []machineUtilisation{}
	for rows.Next() {
		var u machineUtilisation
		if err := rows.Scan(&u.Machine, &u.TotalHours, &u.TotalDepreciation); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		list = append(list, u)
	}
	writeList(w, list, len(list))
}

// --- filament consumption ---

type filamentUsageRow struct {
	Name       string  `json:"name"`
	Grams      float64 `json:"grams"`
	Cost       float64 `json:"cost"`
	QuoteCount int     `json:"quote_count"`
}

// filamentUsage aggregates grams/cost/quote-count per filament across the
// main, support and prototype roles a filament can play in a quote. Cost is
// computed from each filament's current price per gram (not the quote's
// stored aggregate cost_filament, which can't be split back out by role).
func (a *API) filamentUsage() ([]filamentUsageRow, error) {
	filaments, err := a.DB.Query(`SELECT id, name, spool_price, spool_weight_kg FROM filaments`)
	if err != nil {
		return nil, err
	}
	defer filaments.Close()

	type filamentInfo struct {
		name      string
		pricePerG float64
	}
	info := map[int64]*filamentInfo{}
	for filaments.Next() {
		var id int64
		var name string
		var price, weightKg float64
		if err := filaments.Scan(&id, &name, &price, &weightKg); err != nil {
			return nil, err
		}
		pricePerG := 0.0
		if weightKg > 0 {
			pricePerG = price / (weightKg * 1000)
		}
		info[id] = &filamentInfo{name: name, pricePerG: pricePerG}
	}

	usageRows, err := a.DB.Query(
		`SELECT filament_id, print_weight_g, id FROM quotes WHERE status != 'cancelled'
		 UNION ALL
		 SELECT support_filament_id, support_weight_g, id FROM quotes WHERE status != 'cancelled' AND supports_enabled = 1 AND support_filament_id IS NOT NULL
		 UNION ALL
		 SELECT prototype_filament_id, prototype_weight_g, id FROM quotes WHERE status != 'cancelled' AND prototypes_enabled = 1 AND prototype_filament_id IS NOT NULL`,
	)
	if err != nil {
		return nil, err
	}
	defer usageRows.Close()

	type agg struct {
		grams  float64
		quotes map[int64]bool
	}
	aggs := map[int64]*agg{}
	for usageRows.Next() {
		var filamentID, quoteID int64
		var grams float64
		if err := usageRows.Scan(&filamentID, &grams, &quoteID); err != nil {
			return nil, err
		}
		bucket, ok := aggs[filamentID]
		if !ok {
			bucket = &agg{quotes: map[int64]bool{}}
			aggs[filamentID] = bucket
		}
		bucket.grams += grams
		bucket.quotes[quoteID] = true
	}

	result := []filamentUsageRow{}
	for filamentID, bucket := range aggs {
		fi, ok := info[filamentID]
		if !ok {
			continue
		}
		result = append(result, filamentUsageRow{
			Name:       fi.name,
			Grams:      bucket.grams,
			Cost:       bucket.grams * fi.pricePerG,
			QuoteCount: len(bucket.quotes),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Grams > result[j].Grams })
	return result, nil
}

func (a *API) reportFilamentConsumption(w http.ResponseWriter, r *http.Request) {
	usage, err := a.filamentUsage()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeList(w, usage, len(usage))
}
