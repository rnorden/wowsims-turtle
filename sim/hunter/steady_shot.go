package hunter

import (
	"time"

	"github.com/isfir/wowsims-turtle/sim/core"
	"github.com/isfir/wowsims-turtle/sim/core/proto"
)

// Turtle WoW Steady Shot. Ranks 1-5 deal a percentage of weapon damage plus a flat amount;
// ranks 6-8 deal full normalized weapon damage plus a flat amount.
func (hunter *Hunter) getSteadyShotConfig(rank int) core.SpellConfig {
	spellId := [9]int32{0, 3035, 3036, 3037, 3038, 3668, 45970, 45972, 45974}[rank]
	baseDamage := [9]float64{0, 10, 20, 30, 40, 50, 75, 100, 125}[rank]
	weaponMultiplier := [9]float64{0, 0.6, 0.7, 0.8, 0.9, 1, 1, 1, 1}[rank]
	level := [9]int{0, 4, 14, 22, 30, 38, 38, 38, 60}[rank]

	return core.SpellConfig{
		SpellCode:     SpellCode_HunterSteadyShot,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeRanged,
		ProcMask:      core.ProcMaskRangedSpecial,
		Flags:         core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagShot,
		CastType:      proto.CastType_CastTypeRanged,
		Rank:          rank,
		RequiredLevel: level,
		MissileSpeed:  40,

		ManaCost: core.ManaCostOptions{
			BaseCost: 0.05,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Second,
			},
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				cast.CastTime = spell.CastTime()
			},
			IgnoreHaste: true,
			CastTime:    hunter.shotCastTime,
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget >= core.MinRangedAttackDistance
		},

		CritDamageBonus: hunter.mortalShots(),

		DamageMultiplier: 1 + 0.05*float64(hunter.Talents.ImprovedMarksmanship),
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			normalized := rank >= 6
			baseDamage := weaponMultiplier*hunter.rangedWeaponDamage(sim, spell, target, normalized) + baseDamage

			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeRangedHitAndCrit)
			spell.WaitTravelTime(sim, func(s *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	}
}

func (hunter *Hunter) registerSteadyShotSpell() {
	for rank := 1; rank <= 8; rank++ {
		config := hunter.getSteadyShotConfig(rank)

		if config.RequiredLevel <= int(hunter.Level) {
			hunter.SteadyShot = hunter.GetOrRegisterSpell(config)
		}
	}
}

// Extra Shot (51519) is the additional normal ranged attack fired by Endless Quiver.
func (hunter *Hunter) registerExtraShotSpell() {
	if hunter.Talents.EndlessQuiver == 0 || !hunter.HasRangedWeapon() {
		return
	}

	hunter.ExtraShot = hunter.RegisterSpell(core.SpellConfig{
		SpellCode:    SpellCode_HunterExtraShot,
		ActionID:     core.ActionID{SpellID: 51519},
		SpellSchool:  core.SpellSchoolPhysical,
		DefenseType:  core.DefenseTypeRanged,
		ProcMask:     core.ProcMaskRangedAuto,
		Flags:        core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,
		CastType:     proto.CastType_CastTypeRanged,
		MissileSpeed: 40,

		BonusCritRating: hunter.AutoAttacks.RangedConfig().BonusCritRating,
		CritDamageBonus: hunter.mortalShots(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := hunter.rangedWeaponDamage(sim, spell, target, false)
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeRangedHitAndCrit)

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})
}
