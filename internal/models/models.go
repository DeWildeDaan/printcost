package models

// Settings holds the global defaults, keyed the same way as the settings table.
type Settings struct {
	EnergyCost  float64 `json:"energy_cost"`
	LaborRate   float64 `json:"labor_rate"`
	FailureRate float64 `json:"failure_rate"`
	Markup      float64 `json:"markup"`
	Currency    string  `json:"currency"`
	CompanyName string  `json:"company_name"`
}

type Machine struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	TypeTag            string  `json:"type_tag"`
	PurchasePrice      float64 `json:"purchase_price"`
	LifespanHours      float64 `json:"lifespan_hours"`
	ServiceCost        float64 `json:"service_cost"`
	PowerKw            float64 `json:"power_kw"`
	Notes              string  `json:"notes"`
	HourlyDepreciation float64 `json:"hourly_depreciation"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

type BuildPlate struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	PurchasePrice float64 `json:"purchase_price"`
	LifespanHours float64 `json:"lifespan_hours"`
	Notes         string  `json:"notes"`
	HourlyCost    float64 `json:"hourly_cost"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type Nozzle struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Material      string  `json:"material"`
	DiameterMm    float64 `json:"diameter_mm"`
	PurchasePrice float64 `json:"purchase_price"`
	LifespanHours float64 `json:"lifespan_hours"`
	Notes         string  `json:"notes"`
	HourlyCost    float64 `json:"hourly_cost"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type AMSUnit struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	PurchasePrice float64 `json:"purchase_price"`
	LifespanHours float64 `json:"lifespan_hours"`
	ServiceCost   float64 `json:"service_cost"`
	PowerKw       float64 `json:"power_kw"`
	Notes         string  `json:"notes"`
	HourlyCost    float64 `json:"hourly_cost"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// OtherEquipment covers ancillary equipment such as a filament dry cabinet:
// anything with a lifespan/power draw that isn't a printer, plate, nozzle or AMS.
type OtherEquipment struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	PurchasePrice float64 `json:"purchase_price"`
	LifespanHours float64 `json:"lifespan_hours"`
	ServiceCost   float64 `json:"service_cost"`
	PowerKw       float64 `json:"power_kw"`
	Notes         string  `json:"notes"`
	HourlyCost    float64 `json:"hourly_cost"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type Filament struct {
	ID                 int64    `json:"id"`
	Name               string   `json:"name"`
	Brand              string   `json:"brand"`
	Material           string   `json:"material"`
	DiameterMm         float64  `json:"diameter_mm"`
	SpoolPrice         float64  `json:"spool_price"`
	SpoolWeightKg      float64  `json:"spool_weight_kg"`
	DensityGcm3        float64  `json:"density_gcm3"`
	DryingTempC        *int64   `json:"drying_temp_c"`
	DryingTimeHours    *float64 `json:"drying_time_hours"`
	RequiresDryCabinet bool     `json:"requires_dry_cabinet"`
	Notes              string   `json:"notes"`
	PricePerKg         float64  `json:"price_per_kg"`
	PricePerG          float64  `json:"price_per_g"`
	LengthPerRollM     float64  `json:"length_per_roll_m"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

type Consumable struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	UnitLabel string  `json:"unit_label"`
	UnitCost  float64 `json:"unit_cost"`
	Notes     string  `json:"notes"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type ProcessingStep struct {
	ID      int64   `json:"id,omitempty"`
	Label   string  `json:"label"`
	Minutes float64 `json:"minutes"`
}

type QuoteConsumable struct {
	ID           int64   `json:"id,omitempty"`
	ConsumableID int64   `json:"consumable_id"`
	Qty          float64 `json:"qty"`
	UnitCost     float64 `json:"unit_cost"`
}

// Quote is the full read/write representation of a print job quote,
// including nested processing steps and consumables.
type Quote struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	ClientName  string  `json:"client_name"`
	ClientEmail string  `json:"client_email"`
	Notes       string  `json:"notes"`
	ValidUntil  *string `json:"valid_until"`

	MachineID    *int64 `json:"machine_id"`
	BuildPlateID *int64 `json:"build_plate_id"`
	NozzleID     *int64 `json:"nozzle_id"`
	AMSID        *int64 `json:"ams_id"`
	DryCabinetID *int64 `json:"dry_cabinet_id"`

	AMSUsedForPrinting bool `json:"ams_used_for_printing"`

	FilamentID   int64   `json:"filament_id"`
	PrintWeightG float64 `json:"print_weight_g"`
	PrintTimeMin float64 `json:"print_time_min"`

	SupportsEnabled   bool    `json:"supports_enabled"`
	SupportFilamentID *int64  `json:"support_filament_id"`
	SupportWeightG    float64 `json:"support_weight_g"`

	PrototypesEnabled   bool    `json:"prototypes_enabled"`
	PrototypeFilamentID *int64  `json:"prototype_filament_id"`
	PrototypeCount      float64 `json:"prototype_count"`
	PrototypeWeightG    float64 `json:"prototype_weight_g"`
	PrototypeTimeMin    float64 `json:"prototype_time_min"`

	DryingEnabled bool    `json:"drying_enabled"`
	DryingHours   float64 `json:"drying_hours"`

	AdditionalCost     float64 `json:"additional_cost"`
	AdditionalCostNote string  `json:"additional_cost_note"`
	MarkupPct          float64 `json:"markup_pct"`

	EnergyCostSnapshot  float64 `json:"energy_cost_snapshot"`
	LaborRateSnapshot   float64 `json:"labor_rate_snapshot"`
	FailureRateSnapshot float64 `json:"failure_rate_snapshot"`

	CostFilament     float64 `json:"cost_filament"`
	CostElectricity  float64 `json:"cost_electricity"`
	CostDepreciation float64 `json:"cost_depreciation"`
	CostLabor        float64 `json:"cost_labor"`
	CostConsumables  float64 `json:"cost_consumables"`
	CostFailure      float64 `json:"cost_failure"`
	CostMarkup       float64 `json:"cost_markup"`
	CostTotal        float64 `json:"cost_total"`
	SuggestedPrice   float64 `json:"suggested_price"`

	ProcessingSteps []ProcessingStep  `json:"processing_steps"`
	Consumables     []QuoteConsumable `json:"consumables"`

	// Populated on read for display convenience (names of linked records).
	FilamentName          string `json:"filament_name,omitempty"`
	SupportFilamentName   string `json:"support_filament_name,omitempty"`
	PrototypeFilamentName string `json:"prototype_filament_name,omitempty"`
	MachineName           string `json:"machine_name,omitempty"`
	BuildPlateName        string `json:"build_plate_name,omitempty"`
	NozzleName            string `json:"nozzle_name,omitempty"`
	AMSName               string `json:"ams_name,omitempty"`
	DryCabinetName        string `json:"dry_cabinet_name,omitempty"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
