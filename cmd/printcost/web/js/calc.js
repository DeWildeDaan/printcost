import { fmtMoney } from "./utils.js";
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

  const filamentLines = [];
  let costFilament = input.printWeightG * pricePerGram(input.mainFilament);
  if (input.mainFilament && input.printWeightG) {
    filamentLines.push({ label: `${input.mainFilament.name}: ${input.printWeightG}g × ${fmtMoney(pricePerGram(input.mainFilament))}/g`, value: costFilament });
  }
  if (input.supportsEnabled) {
    const v = input.supportWeightG * pricePerGram(input.supportFilament);
    costFilament += v;
    if (input.supportFilament && input.supportWeightG) {
      filamentLines.push({ label: `${input.supportFilament.name} (support): ${input.supportWeightG}g × ${fmtMoney(pricePerGram(input.supportFilament))}/g`, value: v });
    }
  }
  if (input.prototypesEnabled) {
    const v = input.prototypeWeightG * pricePerGram(input.prototypeFilament);
    costFilament += v;
    if (input.prototypeFilament && input.prototypeWeightG) {
      filamentLines.push({ label: `${input.prototypeFilament.name} (prototypes): ${input.prototypeWeightG}g × ${fmtMoney(pricePerGram(input.prototypeFilament))}/g`, value: v });
    }
  }

  let costDepreciation = 0;
  let costElectricity = 0;
  const depreciationLines = [];
  const electricityLines = [];
  const machineHours = printTimeHours + prototypeTimeHours;

  if (input.machine && input.machine.lifespan_hours > 0) {
    const hourlyDep = (input.machine.purchase_price + input.machine.service_cost) / input.machine.lifespan_hours;
    const dep = hourlyDep * machineHours;
    const elec = input.machine.power_kw * machineHours * input.settings.energy_cost;
    costDepreciation += dep;
    costElectricity += elec;
    depreciationLines.push({ label: `${input.machine.name}: ${machineHours.toFixed(2)}h × ${fmtMoney(hourlyDep)}/h`, value: dep });
    electricityLines.push({ label: `${input.machine.name}: ${machineHours.toFixed(2)}h × ${input.machine.power_kw}kW × ${fmtMoney(input.settings.energy_cost)}/kWh`, value: elec });
  }
  if (input.nozzle && input.nozzle.lifespan_hours > 0) {
    const dep = (input.nozzle.purchase_price / input.nozzle.lifespan_hours) * machineHours;
    costDepreciation += dep;
    depreciationLines.push({ label: `${input.nozzle.name}: ${machineHours.toFixed(2)}h × ${fmtMoney(input.nozzle.purchase_price / input.nozzle.lifespan_hours)}/h`, value: dep });
  }
  if (input.buildPlate && input.buildPlate.lifespan_hours > 0) {
    const dep = (input.buildPlate.purchase_price / input.buildPlate.lifespan_hours) * machineHours;
    costDepreciation += dep;
    depreciationLines.push({ label: `${input.buildPlate.name}: ${machineHours.toFixed(2)}h × ${fmtMoney(input.buildPlate.purchase_price / input.buildPlate.lifespan_hours)}/h`, value: dep });
  }

  // Drying happens in the dry cabinet OR the AMS, never both: the cabinet is
  // used when one is selected and a filament in the quote requires it,
  // otherwise drying falls back to the AMS.
  const usingDryCabinet = !!(input.dryingEnabled && input.dryCabinet && input.dryCabinet.lifespan_hours > 0 &&
    anyRequiresDryCabinet(input.mainFilament, input.supportFilament, input.prototypeFilament));

  if (input.ams && input.ams.lifespan_hours > 0) {
    const amsPrintHours = input.amsUsedForPrinting ? printTimeHours + prototypeTimeHours : 0;
    const amsDryHours = input.dryingEnabled && !usingDryCabinet ? input.dryingHours : 0;
    const amsTotalHours = amsPrintHours + amsDryHours;
    const amsHourly = (input.ams.purchase_price + input.ams.service_cost) / input.ams.lifespan_hours;
    const dep = amsHourly * amsTotalHours;
    const elec = input.ams.power_kw * amsTotalHours * input.settings.energy_cost;
    costDepreciation += dep;
    costElectricity += elec;
    if (amsTotalHours > 0) {
      depreciationLines.push({ label: `${input.ams.name}: ${amsTotalHours.toFixed(2)}h × ${fmtMoney(amsHourly)}/h`, value: dep });
      electricityLines.push({ label: `${input.ams.name}: ${amsTotalHours.toFixed(2)}h × ${input.ams.power_kw}kW × ${fmtMoney(input.settings.energy_cost)}/kWh`, value: elec });
    }
  }

  if (usingDryCabinet) {
    const cabinetHourly = (input.dryCabinet.purchase_price + input.dryCabinet.service_cost) / input.dryCabinet.lifespan_hours;
    const dep = cabinetHourly * input.dryingHours;
    const elec = input.dryCabinet.power_kw * input.dryingHours * input.settings.energy_cost;
    costDepreciation += dep;
    costElectricity += elec;
    depreciationLines.push({ label: `${input.dryCabinet.name}: ${input.dryingHours}h × ${fmtMoney(cabinetHourly)}/h`, value: dep });
    electricityLines.push({ label: `${input.dryCabinet.name}: ${input.dryingHours}h × ${input.dryCabinet.power_kw}kW × ${fmtMoney(input.settings.energy_cost)}/kWh`, value: elec });
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
    breakdown: {
      costFilament: filamentLines,
      costElectricity: electricityLines,
      costDepreciation: depreciationLines,
      costLabor: [{ label: `${input.processingMinutes} min (${(input.processingMinutes / 60).toFixed(2)}h) × ${fmtMoney(input.settings.labor_rate)}/h labor rate`, value: costLabor }],
      costConsumables: [{ label: `Sum of consumable line items`, value: costConsumables }],
      costFailure: [{ label: `Subtotal ${fmtMoney(subtotal)} × ${input.settings.failure_rate}% failure allowance`, value: costFailure }],
      costMarkup: [{ label: `(Subtotal + failure) ${fmtMoney(subtotal + costFailure)} × ${input.markupPct || 0}% markup`, value: costMarkup }],
    },
  };
}
