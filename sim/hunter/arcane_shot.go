package hunter

import (
	"time"

	"github.com/isfir/wowsims-turtle/sim/core"
	"github.com/isfir/wowsims-turtle/sim/core/proto"
)

// Turtle WoW Arcane Shot deals a percentage of ranged weapon damage plus flat Arcane damage.
// The flat Arcane part scales with Arcane + generic spell power. Coefficients are the
// effectBonusCoefficient1 values from the Turtle 1.17.1 server spell_template (pre-rework
// spell; 0.429 = 1.5/3.5 from rank 3 on). Turtle's 2026-01-23 rework that added the
// ranged-weapon scaling states "Existing Spell Power scaling is unchanged".
func (hunter *Hunter) getArcaneShotConfig(rank int, timer *core.Timer) core.SpellConfig {
	spellId := [9]int32{0, 3044, 14281, 14282, 14283, 14284, 14285, 14286, 14287}[rank]
	baseDamage := [9]float64{0, 10, 17, 29, 48, 71, 101, 129, 166}[rank]
	weaponMultiplier := [9]float64{0, .10, .15, .20, .25, .30, .35, .40, .45}[rank]
	spellCoeff := [9]float64{0, .204, .3, .429, .429, .429, .429, .429, .429}[rank]
	manaCost := [9]float64{0, 20, 25, 35, 55, 70, 90, 105, 125}[rank]
	level := [9]int{0, 6, 12, 20, 28, 36, 44, 52, 60}[rank]

	return core.SpellConfig{
		SpellCode:     SpellCode_HunterArcaneShot,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolArcane,
		DefenseType:   core.DefenseTypeRanged,
		ProcMask:      core.ProcMaskRangedSpecial,
		Flags:         core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagShot,
		CastType:      proto.CastType_CastTypeRanged,
		Rank:          rank,
		RequiredLevel: level,
		MissileSpeed:  40,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    timer,
				Duration: time.Second*6 - time.Millisecond*500*time.Duration(hunter.Talents.Swiftshot),
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget >= core.MinRangedAttackDistance
		},

		CritDamageBonus: hunter.mortalShots(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			damage := weaponMultiplier*hunter.rangedWeaponDamage(sim, spell, target, false) + baseDamage
			result := spell.CalcDamage(sim, target, damage, spell.OutcomeRangedHitAndCrit)

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	}
}

func (hunter *Hunter) registerArcaneShotSpell() {
	timer := hunter.NewTimer()

	for rank := 1; rank <= 8; rank++ {
		config := hunter.getArcaneShotConfig(rank, timer)

		if config.RequiredLevel <= int(hunter.Level) {
			hunter.ArcaneShot = hunter.GetOrRegisterSpell(config)
		}
	}
}
