package calc

import (
	"math"
	"testing"

	"printcost/internal/models"
)

const eps = 1e-6

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < eps
}

func baseSettings() models.Settings {
	return models.Settings{EnergyCost: 0.28, LaborRate: 20, FailureRate: 5, Markup: 20}
}

func filament(spoolPrice, spoolWeightKg float64, requiresDryCabinet bool) *models.Filament {
	return &models.Filament{SpoolPrice: spoolPrice, SpoolWeightKg: spoolWeightKg, RequiresDryCabinet: requiresDryCabinet}
}

func TestBasicQuoteNoOptions(t *testing.T) {
	settings := baseSettings()
	machine := &models.Machine{PurchasePrice: 1000, ServiceCost: 0, LifespanHours: 1000, PowerKw: 0.2}
	f := filament(25, 1, false) // 0.025 €/g

	in := Input{
		Settings:     settings,
		Machine:      machine,
		MainFilament: f,
		PrintWeightG: 100, // 2.5 filament cost
		PrintTimeMin: 120, // 2h
		MarkupPct:    20,
	}
	r := Calculate(in)

	wantFilament := 100 * 0.025
	wantDep := (1000.0 / 1000.0) * 2 // hourly dep 1 * 2h
	wantElec := 0.2 * 2 * 0.28
	wantSubtotal := wantFilament + wantDep + wantElec
	wantFailure := wantSubtotal * 0.05
	wantMarkup := (wantSubtotal + wantFailure) * 0.20
	wantTotal := wantSubtotal + wantFailure + wantMarkup

	if !almostEqual(r.CostFilament, wantFilament) {
		t.Errorf("CostFilament = %v, want %v", r.CostFilament, wantFilament)
	}
	if !almostEqual(r.CostDepreciation, wantDep) {
		t.Errorf("CostDepreciation = %v, want %v", r.CostDepreciation, wantDep)
	}
	if !almostEqual(r.CostElectricity, wantElec) {
		t.Errorf("CostElectricity = %v, want %v", r.CostElectricity, wantElec)
	}
	if !almostEqual(r.CostTotal, wantTotal) {
		t.Errorf("CostTotal = %v, want %v", r.CostTotal, wantTotal)
	}
	if !almostEqual(r.SuggestedPrice, r.CostTotal) {
		t.Errorf("SuggestedPrice must equal CostTotal")
	}
}

func TestSupportsAndPrototypesAddFilamentCost(t *testing.T) {
	settings := baseSettings()
	main := filament(20, 1, false)    // 0.02 €/g
	support := filament(10, 1, false) // 0.01 €/g
	proto := filament(30, 1, false)   // 0.03 €/g

	in := Input{
		Settings:     settings,
		MainFilament: main,
		PrintWeightG: 50, // 1.0
		PrintTimeMin: 60,

		SupportsEnabled: true,
		SupportFilament: support,
		SupportWeightG:  20, // 0.2

		PrototypesEnabled: true,
		PrototypeFilament: proto,
		PrototypeWeightG:  10, // 0.3
		PrototypeTimeMin:  30,

		MarkupPct: 0,
	}
	r := Calculate(in)
	want := 50*0.02 + 20*0.01 + 10*0.03
	if !almostEqual(r.CostFilament, want) {
		t.Errorf("CostFilament = %v, want %v", r.CostFilament, want)
	}
}

func TestAMSUsedForPrintingToggle(t *testing.T) {
	settings := baseSettings()
	main := filament(20, 1, false)
	ams := &models.AMSUnit{PurchasePrice: 300, ServiceCost: 0, LifespanHours: 3000, PowerKw: 0.05}

	base := Input{
		Settings:     settings,
		MainFilament: main,
		PrintWeightG: 10,
		PrintTimeMin: 120, // 2h
		AMS:          ams,
		MarkupPct:    0,
	}

	on := base
	on.AMSUsedForPrinting = true
	rOn := Calculate(on)

	off := base
	off.AMSUsedForPrinting = false
	rOff := Calculate(off)

	amsHourly := 300.0 / 3000.0
	wantOnDep := amsHourly * 2 // 2h print time
	wantOnElec := 0.05 * 2 * 0.28

	if !almostEqual(rOn.CostDepreciation, wantOnDep) {
		t.Errorf("toggle on: CostDepreciation = %v, want %v", rOn.CostDepreciation, wantOnDep)
	}
	if !almostEqual(rOn.CostElectricity, wantOnElec) {
		t.Errorf("toggle on: CostElectricity = %v, want %v", rOn.CostElectricity, wantOnElec)
	}
	if !almostEqual(rOff.CostDepreciation, 0) {
		t.Errorf("toggle off: CostDepreciation = %v, want 0", rOff.CostDepreciation)
	}
	if !almostEqual(rOff.CostElectricity, 0) {
		t.Errorf("toggle off: CostElectricity = %v, want 0", rOff.CostElectricity)
	}
}

func TestAMSDryingIsIndependentOfPrintingToggle(t *testing.T) {
	settings := baseSettings()
	main := filament(20, 1, false)
	ams := &models.AMSUnit{PurchasePrice: 300, ServiceCost: 0, LifespanHours: 3000, PowerKw: 0.05}

	in := Input{
		Settings:           settings,
		MainFilament:       main,
		PrintWeightG:       10,
		PrintTimeMin:       60,
		AMS:                ams,
		AMSUsedForPrinting: false, // AMS not used during printing...
		DryingEnabled:      true,
		DryingHours:        4, // ...but drying still uses it
		MarkupPct:          0,
	}
	r := Calculate(in)

	amsHourly := 300.0 / 3000.0
	wantDep := amsHourly * 4
	wantElec := 0.05 * 4 * 0.28

	if !almostEqual(r.CostDepreciation, wantDep) {
		t.Errorf("CostDepreciation = %v, want %v", r.CostDepreciation, wantDep)
	}
	if !almostEqual(r.CostElectricity, wantElec) {
		t.Errorf("CostElectricity = %v, want %v", r.CostElectricity, wantElec)
	}
}

func TestDryCabinetOnlyAppliesWhenFilamentRequiresIt(t *testing.T) {
	settings := baseSettings()
	cabinet := &models.OtherEquipment{PurchasePrice: 200, ServiceCost: 0, LifespanHours: 2000, PowerKw: 0.1}

	requiring := filament(20, 1, true)
	notRequiring := filament(20, 1, false)

	base := Input{
		Settings:      settings,
		PrintWeightG:  10,
		PrintTimeMin:  60,
		DryingEnabled: true,
		DryingHours:   3,
		DryCabinet:    cabinet,
		MarkupPct:     0,
	}

	withReq := base
	withReq.MainFilament = requiring
	rWith := Calculate(withReq)

	without := base
	without.MainFilament = notRequiring
	rWithout := Calculate(without)

	cabinetHourly := 200.0 / 2000.0
	wantDep := cabinetHourly * 3
	wantElec := 0.1 * 3 * 0.28

	if !almostEqual(rWith.CostDepreciation, wantDep) {
		t.Errorf("CostDepreciation = %v, want %v", rWith.CostDepreciation, wantDep)
	}
	if !almostEqual(rWith.CostElectricity, wantElec) {
		t.Errorf("CostElectricity = %v, want %v", rWith.CostElectricity, wantElec)
	}
	if !almostEqual(rWithout.CostDepreciation, 0) {
		t.Errorf("without requirement: CostDepreciation = %v, want 0", rWithout.CostDepreciation)
	}
	if !almostEqual(rWithout.CostElectricity, 0) {
		t.Errorf("without requirement: CostElectricity = %v, want 0", rWithout.CostElectricity)
	}
}

func TestDryingUsesCabinetOrAMSNeverBoth(t *testing.T) {
	settings := baseSettings()
	ams := &models.AMSUnit{PurchasePrice: 300, ServiceCost: 0, LifespanHours: 3000, PowerKw: 0.05}
	cabinet := &models.OtherEquipment{PurchasePrice: 200, ServiceCost: 0, LifespanHours: 2000, PowerKw: 0.1}
	requiring := filament(20, 1, true)

	in := Input{
		Settings:           settings,
		MainFilament:       requiring,
		PrintWeightG:       10,
		PrintTimeMin:       60,
		AMS:                ams,
		AMSUsedForPrinting: false,
		DryCabinet:         cabinet,
		DryingEnabled:      true,
		DryingHours:        4,
		MarkupPct:          0,
	}
	r := Calculate(in)

	cabinetHourly := 200.0 / 2000.0
	wantDep := cabinetHourly * 4 // only the cabinet, not the AMS too
	wantElec := 0.1 * 4 * 0.28

	if !almostEqual(r.CostDepreciation, wantDep) {
		t.Errorf("CostDepreciation = %v, want %v (cabinet only, AMS must not also charge for drying)", r.CostDepreciation, wantDep)
	}
	if !almostEqual(r.CostElectricity, wantElec) {
		t.Errorf("CostElectricity = %v, want %v (cabinet only, AMS must not also charge for drying)", r.CostElectricity, wantElec)
	}
}

func TestFailureAndMarkupOrdering(t *testing.T) {
	// markup must be applied to (subtotal + failure), not just subtotal.
	settings := models.Settings{EnergyCost: 0, LaborRate: 0, FailureRate: 10, Markup: 0}
	f := filament(100, 1, false) // 0.1 €/g

	in := Input{
		Settings:     settings,
		MainFilament: f,
		PrintWeightG: 100, // subtotal = 10
		MarkupPct:    50,
	}
	r := Calculate(in)

	subtotal := 10.0
	wantFailure := subtotal * 0.10                   // 1.0
	wantMarkup := (subtotal + wantFailure) * 0.50    // 5.5
	wantTotal := subtotal + wantFailure + wantMarkup // 16.5

	if !almostEqual(r.CostFailure, wantFailure) {
		t.Errorf("CostFailure = %v, want %v", r.CostFailure, wantFailure)
	}
	if !almostEqual(r.CostMarkup, wantMarkup) {
		t.Errorf("CostMarkup = %v, want %v", r.CostMarkup, wantMarkup)
	}
	if !almostEqual(r.CostTotal, wantTotal) {
		t.Errorf("CostTotal = %v, want %v", r.CostTotal, wantTotal)
	}
}

func TestPrototypeTimeCountsTowardMachineDepreciationAndElectricity(t *testing.T) {
	settings := baseSettings()
	machine := &models.Machine{PurchasePrice: 1000, ServiceCost: 0, LifespanHours: 1000, PowerKw: 0.2}
	nozzle := &models.Nozzle{PurchasePrice: 20, LifespanHours: 200}
	plate := &models.BuildPlate{PurchasePrice: 30, LifespanHours: 300}
	main := filament(20, 1, false)
	proto := filament(20, 1, false)

	in := Input{
		Settings:     settings,
		Machine:      machine,
		Nozzle:       nozzle,
		BuildPlate:   plate,
		MainFilament: main,
		PrintWeightG: 10,
		PrintTimeMin: 60, // 1h

		PrototypesEnabled: true,
		PrototypeFilament: proto,
		PrototypeWeightG:  10,
		PrototypeTimeMin:  30, // 0.5h

		MarkupPct: 0,
	}
	r := Calculate(in)

	machineHours := 1.5 // print + prototype time
	wantDep := (1000.0/1000.0)*machineHours + (20.0/200.0)*machineHours + (30.0/300.0)*machineHours
	wantElec := 0.2 * machineHours * 0.28

	if !almostEqual(r.CostDepreciation, wantDep) {
		t.Errorf("CostDepreciation = %v, want %v (should include prototype print time)", r.CostDepreciation, wantDep)
	}
	if !almostEqual(r.CostElectricity, wantElec) {
		t.Errorf("CostElectricity = %v, want %v (should include prototype print time)", r.CostElectricity, wantElec)
	}
}

func TestBuildPlateHourlyCost(t *testing.T) {
	settings := baseSettings()
	plate := &models.BuildPlate{PurchasePrice: 30, LifespanHours: 300}
	f := filament(20, 1, false)

	in := Input{
		Settings:     settings,
		BuildPlate:   plate,
		MainFilament: f,
		PrintWeightG: 1,
		PrintTimeMin: 600, // 10h
		MarkupPct:    0,
	}
	r := Calculate(in)
	want := (30.0 / 300.0) * 10 // hourly rate * print hours
	if !almostEqual(r.CostDepreciation, want) {
		t.Errorf("CostDepreciation = %v, want %v (hourly, scales with print time)", r.CostDepreciation, want)
	}
}
