package hunter

import (
	"github.com/isfir/wowsims-turtle/sim/core"
)

// Carve (Survival talent): a sweeping melee attack hitting up to 5 enemies for a percentage of
// main hand weapon damage. Shares its cooldown with Multi-Shot.
func (hunter *Hunter) registerCarveSpell(multiShotTimer *core.Timer) {
	if !hunter.Talents.Carve {
		return
	}

	rank := 0
	for r := 3; r >= 1; r-- {
		if []int{0, 20, 38, 56}[r] <= int(hunter.Level) {
			rank = r
			break
		}
	}
	if rank == 0 {
		return
	}

	spellID := [4]int32{0, 51575, 52415, 52416}[rank]
	weaponMultiplier := [4]float64{0, .60, .65, .70}[rank]
	maxTargets := min(5, hunter.Env.GetNumTargets())
	results := make([]*core.SpellResult, maxTargets)

	hunter.Carve = hunter.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_HunterCarve,
		ActionID:    core.ActionID{SpellID: spellID},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagStrike,
		Rank:        rank,

		ManaCost: core.ManaCostOptions{
			BaseCost: 0.08,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    multiShotTimer,
				Duration: multiShotCooldown,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget <= core.MaxMeleeAttackDistance
		},

		BonusCritRating:  hunter.savageStrikesCritRating(),
		CritDamageBonus:  hunter.killerInstinctCritDamageBonus(),
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			curTarget := target
			for i := int32(0); i < maxTargets; i++ {
				damage := weaponMultiplier * spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(curTarget))
				results[i] = spell.CalcDamage(sim, curTarget, damage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
				curTarget = sim.Environment.NextTargetUnit(curTarget)
			}
			for i := int32(0); i < maxTargets; i++ {
				spell.DealDamage(sim, results[i])
			}
		},
	})
}
