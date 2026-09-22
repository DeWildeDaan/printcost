// Mirrors internal/calc/calc.go. Used only for the live preview in the
// quote builder — the server recomputes authoritatively on save.

function pricePerGram(f) {
  if (!f || !f.spool_weight_kg) return 0;
  return f.spool_price / (f.spool_weight_kg * 1000);
}

function anyRequiresDryCabinet(...fs) {
  return fs.some((f) => f && f.requires_dry_cabinet);
}

export function calculate(input) {
  const printTimeHours = input.printTimeMin / 60;
  const prototypeTimeHours = input.prototypesEnabled ? input.prototypeTimeMin / 60 : 0;

  let costFilament = input.printWeightG * pricePerGram(input.mainFilament);
  if (input.supportsEnabled) {
    costFilament += input.supportWeightG * pricePerGram(input.supportFilament);
  }
  if (input.prototypesEnabled) {
    costFilament += input.prototypeWeightG * pricePerGram(input.prototypeFilament);
  }

  let costDepreciation = 0;
  let costElectricity = 0;

  if (input.machine && input.machine.lifespan_hours > 0) {
    const hourlyDep = (input.machine.purchase_price + input.machine.service_cost) / input.machine.lifespan_hours;
    costDepreciation += hourlyDep * printTimeHours;
    costElectricity += input.machine.power_kw * printTimeHours * input.settings.energy_cost;
  }
  if (input.nozzle && input.nozzle.lifespan_hours > 0) {
    costDepreciation += (input.nozzle.purchase_price / input.nozzle.lifespan_hours) * printTimeHours;
  }
  if (input.buildPlate && input.buildPlate.lifespan_hours > 0) {
    costDepreciation += (input.buildPlate.purchase_price / input.buildPlate.lifespan_hours) * printTimeHours;
  }

  if (input.ams && input.ams.lifespan_hours > 0) {
    const amsPrintHours = input.amsUsedForPrinting ? printTimeHours + prototypeTimeHours : 0;
    const amsDryHours = input.dryingEnabled ? input.dryingHours : 0;
    const amsTotalHours = amsPrintHours + amsDryHours;
    const amsHourly = (input.ams.purchase_price + input.ams.service_cost) / input.ams.lifespan_hours;
    costDepreciation += amsHourly * amsTotalHours;
    costElectricity += input.ams.power_kw * amsTotalHours * input.settings.energy_cost;
  }

  if (input.dryingEnabled && input.dryCabinet && input.dryCabinet.lifespan_hours > 0) {
    if (anyRequiresDryCabinet(input.mainFilament, input.supportFilament, input.prototypeFilament)) {
      const cabinetHourly = (input.dryCabinet.purchase_price + input.dryCabinet.service_cost) / input.dryCabinet.lifespan_hours;
      costDepreciation += cabinetHourly * input.dryingHours;
      costElectricity += input.dryCabinet.power_kw * input.dryingHours * input.settings.energy_cost;
    }
  }

  const costLabor = (input.processingMinutes / 60) * input.settings.labor_rate;
  const costConsumables = input.consumablesCost;

  const subtotal = costFilament + costDepreciation + costElectricity + costLabor + costConsumables + (input.additionalCost || 0);
  const costFailure = subtotal * (input.settings.failure_rate / 100);
  const costMarkup = (subtotal + costFailure) * ((input.markupPct || 0) / 100);
  const costTotal = subtotal + costFailure + costMarkup;

  return {
    costFilament,
    costElectricity,
    costDepreciation,
    costLabor,
    costConsumables,
    costFailure,
    costMarkup,
    costTotal,
    suggestedPrice: costTotal,
  };
}
