package core

import (
	"testing"

	"github.com/isfir/wowsims-turtle/sim/core/proto"
	"github.com/isfir/wowsims-turtle/sim/core/stats"
)

// OctoWoW 1.17.2 weapon skill rules vs a level 63 (315 defense) target.
func TestAttackTableWeaponSkillOctoWoW(t *testing.T) {
	cases := []struct {
		skill      float64
		miss       float64
		glanceMean float64
	}{
		{300, 0.080, 0.65},
		{305, 0.070, 0.75},
		{308, 0.064, 0.81},
		{315, 0.050, 0.95},
		{320, 0.050, 0.95}, // capped at 315
	}

	weapon := &Item{
		HandType:   proto.HandType_HandTypeMainHand,
		WeaponType: proto.WeaponType_WeaponTypeSword,
	}
	defender := &Unit{Type: EnemyUnit, Level: 63}

	for _, c := range cases {
		attacker := &Unit{Type: PlayerUnit, Level: 60, PseudoStats: stats.NewPseudoStats()}
		attacker.PseudoStats.SwordsSkill = c.skill - 300

		table := NewAttackTable(attacker, defender, weapon)

		if !WithinToleranceFloat64(c.miss, table.BaseMissChance, 1e-9) {
			t.Errorf("skill %v: miss = %v, want %v", c.skill, table.BaseMissChance, c.miss)
		}
		if table.HitSuppression != 0 {
			t.Errorf("skill %v: hit suppression = %v, want 0", c.skill, table.HitSuppression)
		}
		mean := (table.GlanceMultiplierMin + table.GlanceMultiplierMax) / 2
		if !WithinToleranceFloat64(c.glanceMean, mean, 1e-9) {
			t.Errorf("skill %v: glance mean = %v, want %v", c.skill, mean, c.glanceMean)
		}
		if table.GlanceMultiplierMin > table.GlanceMultiplierMax {
			t.Errorf("skill %v: glance min %v > max %v", c.skill, table.GlanceMultiplierMin, table.GlanceMultiplierMax)
		}
	}
}
