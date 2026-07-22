import { cpuDisplayMode } from "@/stores/settings";

/**
 * Picks the CPU value to display based on the user's preference.
 *
 * - "utilization" (default): 0-100 utilization of the available CPU.
 * - "cores": Linux/top style where 100% equals one full CPU core, so a value
 *   can exceed 100 when more than one core is used.
 *
 * @param utilization whole-CPU utilization percentage (0-100)
 * @param perCore per-core value where 100 == one core
 */
export function cpuDisplayValue(utilization: number, perCore: number): number {
  return Math.max(0, cpuDisplayMode.value === "cores" ? perCore : utilization);
}
