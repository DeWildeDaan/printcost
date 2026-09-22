// Package calc implements the print-quote cost formula. It is the single
// source of truth for cost calculation; the frontend's live preview mirrors
// this logic in JS but the server recomputes on save.
package calc

import "printcost/internal/models"

// Input bundles everything needed to price a quote. Equipment/filament
// pointers are nil when that optional selection wasn't made.
type Input struct {
	Settings models.Settings

	Machine    *models.Machine
	BuildPlate *models.BuildPlate
	Nozzle     *models.Nozzle
	AMS        *models.AMSUnit
	DryCabinet *models.OtherEquipment

	AMSUsedForPrinting bool

	MainFilament *models.Filament
	PrintWeightG float64
	PrintTimeMin float64

	SupportsEnabled bool
	SupportFilament *models.Filament
	SupportWeightG  float64

	PrototypesEnabled bool
	PrototypeFilament *models.Filament
	PrototypeWeightG  float64
	PrototypeTimeMin  float64

	DryingEnabled bool
	DryingHours   float64

	ProcessingMinutes float64 // sum of all pre/post-processing step minutes
	ConsumablesCost   float64 // sum of qty * unit_cost across quote consumables

	AdditionalCost float64
	MarkupPct      float64
}

// Result holds every line item the quote record stores.
type Result struct {
	CostFilament     float64
	CostElectricity  float64
	CostDepreciation float64
	CostLabor        float64
	CostConsumables  float64
	CostFailure      float64
	CostMarkup       float64
	CostTotal        float64
	SuggestedPrice   float64
}

func pricePerGram(f *models.Filament) float64 {
	if f == nil || f.SpoolWeightKg <= 0 {
		return 0
	}
	return f.SpoolPrice / (f.SpoolWeightKg * 1000)
}

func filamentRequiresDryCabinet(fs ...*models.Filament) bool {
	for _, f := range fs {
		if f != nil && f.RequiresDryCabinet {
			return true
		}
	}
	return false
}

// Calculate applies the full quote cost formula.
func Calculate(in Input) Result {
	printTimeHours := in.PrintTimeMin / 60
	prototypeTimeHours := 0.0
	if in.PrototypesEnabled {
		prototypeTimeHours = in.PrototypeTimeMin / 60
	}

	// --- Filament ---
	costFilament := in.PrintWeightG * pricePerGram(in.MainFilament)
	if in.SupportsEnabled {
		costFilament += in.SupportWeightG * pricePerGram(in.SupportFilament)
	}
	if in.PrototypesEnabled {
		costFilament += in.PrototypeWeightG * pricePerGram(in.PrototypeFilament)
	}

	// --- Depreciation & electricity from the printer/nozzle/plate ---
	costDepreciation := 0.0
	costElectricity := 0.0

	if in.Machine != nil && in.Machine.LifespanHours > 0 {
		hourlyDep := (in.Machine.PurchasePrice + in.Machine.ServiceCost) / in.Machine.LifespanHours
		costDepreciation += hourlyDep * printTimeHours
		costElectricity += in.Machine.PowerKw * printTimeHours * in.Settings.EnergyCost
	}
	if in.Nozzle != nil && in.Nozzle.LifespanHours > 0 {
		costDepreciation += (in.Nozzle.PurchasePrice / in.Nozzle.LifespanHours) * printTimeHours
	}
	if in.BuildPlate != nil && in.BuildPlate.LifespanHours > 0 {
		costDepreciation += (in.BuildPlate.PurchasePrice / in.BuildPlate.LifespanHours) * printTimeHours
	}

	// --- AMS: used for printing (print+prototype hours, toggleable) and
	// independently during drying (always added if drying is enabled and
	// an AMS is selected). ---
	if in.AMS != nil && in.AMS.LifespanHours > 0 {
		amsPrintHours := 0.0
		if in.AMSUsedForPrinting {
			amsPrintHours = printTimeHours + prototypeTimeHours
		}
		amsDryHours := 0.0
		if in.DryingEnabled {
			amsDryHours = in.DryingHours
		}
		amsTotalHours := amsPrintHours + amsDryHours

		amsHourly := (in.AMS.PurchasePrice + in.AMS.ServiceCost) / in.AMS.LifespanHours
		costDepreciation += amsHourly * amsTotalHours
		costElectricity += in.AMS.PowerKw * amsTotalHours * in.Settings.EnergyCost
	}

	// --- Dry cabinet: only when drying is enabled, a cabinet is selected,
	// and at least one filament used in this quote requires it. ---
	if in.DryingEnabled && in.DryCabinet != nil && in.DryCabinet.LifespanHours > 0 {
		needsCabinet := filamentRequiresDryCabinet(in.MainFilament, in.SupportFilament, in.PrototypeFilament)
		if needsCabinet {
			cabinetHourly := (in.DryCabinet.PurchasePrice + in.DryCabinet.ServiceCost) / in.DryCabinet.LifespanHours
			costDepreciation += cabinetHourly * in.DryingHours
			costElectricity += in.DryCabinet.PowerKw * in.DryingHours * in.Settings.EnergyCost
		}
	}

	// --- Labor & consumables ---
	costLabor := (in.ProcessingMinutes / 60) * in.Settings.LaborRate
	costConsumables := in.ConsumablesCost

	subtotal := costFilament + costDepreciation + costElectricity + costLabor + costConsumables + in.AdditionalCost

	costFailure := subtotal * (in.Settings.FailureRate / 100)
	costMarkup := (subtotal + costFailure) * (in.MarkupPct / 100)
	costTotal := subtotal + costFailure + costMarkup

	return Result{
		CostFilament:     costFilament,
		CostElectricity:  costElectricity,
		CostDepreciation: costDepreciation,
		CostLabor:        costLabor,
		CostConsumables:  costConsumables,
		CostFailure:      costFailure,
		CostMarkup:       costMarkup,
		CostTotal:        costTotal,
		SuggestedPrice:   costTotal,
	}
}
