package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"printcost/internal/calc"
	"printcost/internal/models"
)

// --- single-record lookups used to build a calc.Input ---

func (a *API) getMachine(id *int64) (*models.Machine, error) {
	if id == nil {
		return nil, nil
	}
	m, err := scanMachine(a.DB.QueryRow(`SELECT `+machineCols+` FROM machines WHERE id = ?`, *id))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &m, nil
}

func (a *API) getBuildPlate(id *int64) (*models.BuildPlate, error) {
	if id == nil {
		return nil, nil
	}
	p, err := scanBuildPlate(a.DB.QueryRow(`SELECT `+buildPlateCols+` FROM build_plates WHERE id = ?`, *id))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &p, nil
}

func (a *API) getNozzle(id *int64) (*models.Nozzle, error) {
	if id == nil {
		return nil, nil
	}
	n, err := scanNozzle(a.DB.QueryRow(`SELECT `+nozzleCols+` FROM nozzles WHERE id = ?`, *id))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &n, nil
}

func (a *API) getAMS(id *int64) (*models.AMSUnit, error) {
	if id == nil {
		return nil, nil
	}
	u, err := scanAMS(a.DB.QueryRow(`SELECT `+amsCols+` FROM ams_units WHERE id = ?`, *id))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &u, nil
}

func (a *API) getOtherEquipment(id *int64) (*models.OtherEquipment, error) {
	if id == nil {
		return nil, nil
	}
	e, err := scanOtherEquipment(a.DB.QueryRow(`SELECT `+otherEquipmentCols+` FROM other_equipment WHERE id = ?`, *id))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &e, nil
}

func (a *API) getFilament(id *int64) (*models.Filament, error) {
	if id == nil {
		return nil, nil
	}
	f, err := scanFilament(a.DB.QueryRow(`SELECT `+filamentCols+` FROM filaments WHERE id = ?`, *id))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &f, nil
}

// buildCalcInput loads current settings + referenced equipment/filaments and
// assembles the calc.Input for a quote. Costs are always computed from
// current prices, never from stale values on the incoming struct.
func (a *API) buildCalcInput(q *models.Quote) (calc.Input, error) {
	settings, err := a.loadSettings()
	if err != nil {
		return calc.Input{}, err
	}
	machine, err := a.getMachine(q.MachineID)
	if err != nil {
		return calc.Input{}, err
	}
	plate, err := a.getBuildPlate(q.BuildPlateID)
	if err != nil {
		return calc.Input{}, err
	}
	nozzle, err := a.getNozzle(q.NozzleID)
	if err != nil {
		return calc.Input{}, err
	}
	ams, err := a.getAMS(q.AMSID)
	if err != nil {
		return calc.Input{}, err
	}
	dryCabinet, err := a.getOtherEquipment(q.DryCabinetID)
	if err != nil {
		return calc.Input{}, err
	}
	mainFilament, err := a.getFilament(&q.FilamentID)
	if err != nil {
		return calc.Input{}, err
	}
	var supportFilament *models.Filament
	if q.SupportsEnabled {
		supportFilament, err = a.getFilament(q.SupportFilamentID)
		if err != nil {
			return calc.Input{}, err
		}
	}
	var prototypeFilament *models.Filament
	if q.PrototypesEnabled {
		prototypeFilament, err = a.getFilament(q.PrototypeFilamentID)
		if err != nil {
			return calc.Input{}, err
		}
	}

	processingMinutes := 0.0
	for _, step := range q.ProcessingSteps {
		processingMinutes += step.Minutes
	}
	consumablesCost := 0.0
	for _, c := range q.Consumables {
		consumablesCost += c.Qty * c.UnitCost
	}

	return calc.Input{
		Settings:           settings,
		Machine:            machine,
		BuildPlate:         plate,
		Nozzle:             nozzle,
		AMS:                ams,
		DryCabinet:         dryCabinet,
		AMSUsedForPrinting: q.AMSUsedForPrinting,
		MainFilament:       mainFilament,
		PrintWeightG:       q.PrintWeightG,
		PrintTimeMin:       q.PrintTimeMin,
		SupportsEnabled:    q.SupportsEnabled,
		SupportFilament:    supportFilament,
		SupportWeightG:     q.SupportWeightG,
		PrototypesEnabled:  q.PrototypesEnabled,
		PrototypeFilament:  prototypeFilament,
		PrototypeWeightG:   q.PrototypeWeightG,
		PrototypeTimeMin:   q.PrototypeTimeMin,
		DryingEnabled:      q.DryingEnabled,
		DryingHours:        q.DryingHours,
		ProcessingMinutes:  processingMinutes,
		ConsumablesCost:    consumablesCost,
		AdditionalCost:     q.AdditionalCost,
		MarkupPct:          q.MarkupPct,
	}, nil
}

func validateQuote(q *models.Quote) error {
	if q.Name == "" {
		return fmt.Errorf("name is required")
	}
	if q.FilamentID == 0 {
		return fmt.Errorf("filament_id is required")
	}
	if q.PrintWeightG <= 0 {
		return fmt.Errorf("print_weight_g must be greater than 0")
	}
	if q.PrintTimeMin <= 0 {
		return fmt.Errorf("print_time_min must be greater than 0")
	}
	if q.Status == "" {
		q.Status = "draft"
	}
	validStatuses := map[string]bool{"draft": true, "sent": true, "paid": true, "cancelled": true}
	if !validStatuses[q.Status] {
		return fmt.Errorf("invalid status %q", q.Status)
	}
	return nil
}

// applyCalculatedCosts recomputes and stores the quote's cost fields.
func (a *API) applyCalculatedCosts(q *models.Quote) error {
	input, err := a.buildCalcInput(q)
	if err != nil {
		return err
	}
	result := calc.Calculate(input)

	q.EnergyCostSnapshot = input.Settings.EnergyCost
	q.LaborRateSnapshot = input.Settings.LaborRate
	q.FailureRateSnapshot = input.Settings.FailureRate

	q.CostFilament = result.CostFilament
	q.CostElectricity = result.CostElectricity
	q.CostDepreciation = result.CostDepreciation
	q.CostLabor = result.CostLabor
	q.CostConsumables = result.CostConsumables
	q.CostFailure = result.CostFailure
	q.CostMarkup = result.CostMarkup
	q.CostTotal = result.CostTotal
	q.SuggestedPrice = result.SuggestedPrice
	return nil
}

func (a *API) saveQuote(w http.ResponseWriter, q *models.Quote, isCreate bool) {
	if err := validateQuote(q); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.applyCalculatedCosts(q); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	if isCreate {
		res, err := tx.Exec(insertQuoteSQL, quoteInsertArgs(q)...)
		if err != nil {
			if isForeignKeyError(err) {
				writeError(w, http.StatusBadRequest, "one of the referenced equipment/filament records does not exist")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		q.ID, _ = res.LastInsertId()
	} else {
		args := append(quoteInsertArgs(q), q.ID)
		if _, err := tx.Exec(updateQuoteSQL, args...); err != nil {
			if isForeignKeyError(err) {
				writeError(w, http.StatusBadRequest, "one of the referenced equipment/filament records does not exist")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if _, err := tx.Exec(`DELETE FROM quote_processing_steps WHERE quote_id = ?`, q.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if _, err := tx.Exec(`DELETE FROM quote_consumables WHERE quote_id = ?`, q.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	for i, step := range q.ProcessingSteps {
		if _, err := tx.Exec(
			`INSERT INTO quote_processing_steps (quote_id, label, minutes, sort_order) VALUES (?, ?, ?, ?)`,
			q.ID, step.Label, step.Minutes, i,
		); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	for _, c := range q.Consumables {
		if _, err := tx.Exec(
			`INSERT INTO quote_consumables (quote_id, consumable_id, qty, unit_cost) VALUES (?, ?, ?, ?)`,
			q.ID, c.ConsumableID, c.Qty, c.UnitCost,
		); err != nil {
			if isForeignKeyError(err) {
				writeError(w, http.StatusBadRequest, "one of the selected consumables does not exist")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	full, err := a.loadQuote(q.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	status := http.StatusOK
	if isCreate {
		status = http.StatusCreated
	}
	writeJSON(w, status, full)
}

const insertQuoteSQL = `
	INSERT INTO quotes (
		name, status, client_name, client_email, notes, valid_until,
		machine_id, build_plate_id, nozzle_id, ams_id, dry_cabinet_id, ams_used_for_printing,
		filament_id, print_weight_g, print_time_min,
		supports_enabled, support_filament_id, support_weight_g,
		prototypes_enabled, prototype_filament_id, prototype_count, prototype_weight_g, prototype_time_min,
		drying_enabled, drying_hours,
		additional_cost, additional_cost_note, markup_pct,
		energy_cost_snapshot, labor_rate_snapshot, failure_rate_snapshot,
		cost_filament, cost_electricity, cost_depreciation, cost_labor, cost_consumables,
		cost_failure, cost_markup, cost_total, suggested_price
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const updateQuoteSQL = `
	UPDATE quotes SET
		name=?, status=?, client_name=?, client_email=?, notes=?, valid_until=?,
		machine_id=?, build_plate_id=?, nozzle_id=?, ams_id=?, dry_cabinet_id=?, ams_used_for_printing=?,
		filament_id=?, print_weight_g=?, print_time_min=?,
		supports_enabled=?, support_filament_id=?, support_weight_g=?,
		prototypes_enabled=?, prototype_filament_id=?, prototype_count=?, prototype_weight_g=?, prototype_time_min=?,
		drying_enabled=?, drying_hours=?,
		additional_cost=?, additional_cost_note=?, markup_pct=?,
		energy_cost_snapshot=?, labor_rate_snapshot=?, failure_rate_snapshot=?,
		cost_filament=?, cost_electricity=?, cost_depreciation=?, cost_labor=?, cost_consumables=?,
		cost_failure=?, cost_markup=?, cost_total=?, suggested_price=?,
		updated_at=datetime('now')
	WHERE id=?
`

func quoteInsertArgs(q *models.Quote) []any {
	return []any{
		q.Name, q.Status, q.ClientName, q.ClientEmail, q.Notes, q.ValidUntil,
		q.MachineID, q.BuildPlateID, q.NozzleID, q.AMSID, q.DryCabinetID, q.AMSUsedForPrinting,
		q.FilamentID, q.PrintWeightG, q.PrintTimeMin,
		q.SupportsEnabled, q.SupportFilamentID, q.SupportWeightG,
		q.PrototypesEnabled, q.PrototypeFilamentID, q.PrototypeCount, q.PrototypeWeightG, q.PrototypeTimeMin,
		q.DryingEnabled, q.DryingHours,
		q.AdditionalCost, q.AdditionalCostNote, q.MarkupPct,
		q.EnergyCostSnapshot, q.LaborRateSnapshot, q.FailureRateSnapshot,
		q.CostFilament, q.CostElectricity, q.CostDepreciation, q.CostLabor, q.CostConsumables,
		q.CostFailure, q.CostMarkup, q.CostTotal, q.SuggestedPrice,
	}
}

const quoteSelectCols = `
	q.id, q.name, q.status, q.client_name, q.client_email, q.notes, q.valid_until,
	q.machine_id, q.build_plate_id, q.nozzle_id, q.ams_id, q.dry_cabinet_id, q.ams_used_for_printing,
	q.filament_id, q.print_weight_g, q.print_time_min,
	q.supports_enabled, q.support_filament_id, q.support_weight_g,
	q.prototypes_enabled, q.prototype_filament_id, q.prototype_count, q.prototype_weight_g, q.prototype_time_min,
	q.drying_enabled, q.drying_hours,
	q.additional_cost, q.additional_cost_note, q.markup_pct,
	q.energy_cost_snapshot, q.labor_rate_snapshot, q.failure_rate_snapshot,
	q.cost_filament, q.cost_electricity, q.cost_depreciation, q.cost_labor, q.cost_consumables,
	q.cost_failure, q.cost_markup, q.cost_total, q.suggested_price,
	q.created_at, q.updated_at,
	f.name,
	COALESCE(sf.name, ''), COALESCE(pf.name, ''),
	COALESCE(m.name, ''), COALESCE(bp.name, ''), COALESCE(nz.name, ''), COALESCE(ams.name, ''), COALESCE(dc.name, '')
`

const quoteFromJoins = `
	FROM quotes q
	JOIN filaments f ON f.id = q.filament_id
	LEFT JOIN filaments sf ON sf.id = q.support_filament_id
	LEFT JOIN filaments pf ON pf.id = q.prototype_filament_id
	LEFT JOIN machines m ON m.id = q.machine_id
	LEFT JOIN build_plates bp ON bp.id = q.build_plate_id
	LEFT JOIN nozzles nz ON nz.id = q.nozzle_id
	LEFT JOIN ams_units ams ON ams.id = q.ams_id
	LEFT JOIN other_equipment dc ON dc.id = q.dry_cabinet_id
`

func scanQuote(row interface{ Scan(...any) error }) (models.Quote, error) {
	var q models.Quote
	err := row.Scan(
		&q.ID, &q.Name, &q.Status, &q.ClientName, &q.ClientEmail, &q.Notes, &q.ValidUntil,
		&q.MachineID, &q.BuildPlateID, &q.NozzleID, &q.AMSID, &q.DryCabinetID, &q.AMSUsedForPrinting,
		&q.FilamentID, &q.PrintWeightG, &q.PrintTimeMin,
		&q.SupportsEnabled, &q.SupportFilamentID, &q.SupportWeightG,
		&q.PrototypesEnabled, &q.PrototypeFilamentID, &q.PrototypeCount, &q.PrototypeWeightG, &q.PrototypeTimeMin,
		&q.DryingEnabled, &q.DryingHours,
		&q.AdditionalCost, &q.AdditionalCostNote, &q.MarkupPct,
		&q.EnergyCostSnapshot, &q.LaborRateSnapshot, &q.FailureRateSnapshot,
		&q.CostFilament, &q.CostElectricity, &q.CostDepreciation, &q.CostLabor, &q.CostConsumables,
		&q.CostFailure, &q.CostMarkup, &q.CostTotal, &q.SuggestedPrice,
		&q.CreatedAt, &q.UpdatedAt,
		&q.FilamentName,
		&q.SupportFilamentName, &q.PrototypeFilamentName,
		&q.MachineName, &q.BuildPlateName, &q.NozzleName, &q.AMSName, &q.DryCabinetName,
	)
	return q, err
}

func (a *API) loadQuote(id int64) (*models.Quote, error) {
	row := a.DB.QueryRow(`SELECT `+quoteSelectCols+quoteFromJoins+` WHERE q.id = ?`, id)
	q, err := scanQuote(row)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	steps, err := a.DB.Query(`SELECT id, label, minutes FROM quote_processing_steps WHERE quote_id = ? ORDER BY sort_order`, id)
	if err != nil {
		return nil, err
	}
	defer steps.Close()
	q.ProcessingSteps = []models.ProcessingStep{}
	for steps.Next() {
		var s models.ProcessingStep
		if err := steps.Scan(&s.ID, &s.Label, &s.Minutes); err != nil {
			return nil, err
		}
		q.ProcessingSteps = append(q.ProcessingSteps, s)
	}

	cons, err := a.DB.Query(`SELECT id, consumable_id, qty, unit_cost FROM quote_consumables WHERE quote_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer cons.Close()
	q.Consumables = []models.QuoteConsumable{}
	for cons.Next() {
		var c models.QuoteConsumable
		if err := cons.Scan(&c.ID, &c.ConsumableID, &c.Qty, &c.UnitCost); err != nil {
			return nil, err
		}
		q.Consumables = append(q.Consumables, c)
	}

	return &q, nil
}

func (a *API) listQuotes(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	sortKey := r.URL.Query().Get("sort")

	query := `SELECT ` + quoteSelectCols + quoteFromJoins + ` WHERE 1=1`
	var args []any

	if status != "" && status != "all" {
		query += ` AND q.status = ?`
		args = append(args, status)
	}
	if search != "" {
		query += ` AND (q.name LIKE ? OR q.client_name LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like)
	}

	switch sortKey {
	case "name":
		query += ` ORDER BY q.name ASC`
	case "price":
		query += ` ORDER BY q.suggested_price DESC`
	default:
		query += ` ORDER BY q.created_at DESC`
	}

	rows, err := a.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	list := []models.Quote{}
	for rows.Next() {
		q, err := scanQuote(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		list = append(list, q)
	}
	writeList(w, list, len(list))
}

func (a *API) createQuote(w http.ResponseWriter, r *http.Request) {
	var q models.Quote
	if err := decodeJSON(r, &q); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	a.saveQuote(w, &q, true)
}

func (a *API) getQuote(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	q, err := a.loadQuote(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if q == nil {
		writeError(w, http.StatusNotFound, "quote not found")
		return
	}
	writeJSON(w, http.StatusOK, q)
}

func (a *API) updateQuote(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var q models.Quote
	if err := decodeJSON(r, &q); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	q.ID = id
	a.saveQuote(w, &q, false)
}

func (a *API) deleteQuote(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := a.DB.Exec(`DELETE FROM quotes WHERE id = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) duplicateQuote(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	q, err := a.loadQuote(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if q == nil {
		writeError(w, http.StatusNotFound, "quote not found")
		return
	}
	q.ID = 0
	q.Name = q.Name + " (copy)"
	q.Status = "draft"
	a.saveQuote(w, q, true)
}

func (a *API) patchQuoteStatus(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	validStatuses := map[string]bool{"draft": true, "sent": true, "paid": true, "cancelled": true}
	if !validStatuses[body.Status] {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	res, err := a.DB.Exec(`UPDATE quotes SET status = ?, updated_at = datetime('now') WHERE id = ?`, body.Status, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "quote not found")
		return
	}
	q, err := a.loadQuote(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, q)
}
