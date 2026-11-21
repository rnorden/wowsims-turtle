package core

import (
	"testing"

	"github.com/isfir/wowsims-turtle/sim/core/proto"
	"github.com/isfir/wowsims-turtle/sim/core/stats"
)

// A +2% attack speed item must speed up ranged attacks by exactly 2% (not 2% twice),
// matching melee.
func TestItemAttackSpeedAppliedOnceToRanged(t *testing.T) {
	character := &Character{
		Unit: Unit{
			PseudoStats: stats.NewPseudoStats(),
		},
	}
	for i := range character.itemStatMultipliers {
		character.itemStatMultipliers[i] = 1
	}
	character.Equipment[proto.ItemSlot_ItemSlotHands] = Item{
		ID:    1,
		Stats: stats.Stats{stats.MeleeHaste: 2},
	}

	character.applyEquipment()

	if got := character.GetStat(stats.MeleeHaste); got != 2 {
		t.Fatalf("expected MeleeHaste stat 2, got %v", got)
	}
	if got := character.SwingSpeed(); !WithinToleranceFloat64(1.02, got, 1e-9) {
		t.Fatalf("expected melee swing speed 1.02, got %v", got)
	}
	if got := character.RangedSwingSpeed(); !WithinToleranceFloat64(1.02, got, 1e-9) {
		t.Fatalf("expected ranged swing speed 1.02, got %v (haste applied twice?)", got)
	}
}
