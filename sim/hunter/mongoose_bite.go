package hunter

import (
	"time"

	"github.com/isfir/wowsims-turtle/sim/core"
)

// Turtle WoW Mongoose Bite: no longer requires a dodge, deals a percentage of weapon damage plus a
// flat amount, and strikes with both weapons when dual wielding.
func (hunter *Hunter) getMongooseBiteConfig(rank int) core.SpellConfig {
	spellId := [5]int32{0, 1495, 14269, 14270, 14271}[rank]
	baseDamage := [5]float64{0, 15, 25, 45, 70}[rank]
	weaponMultiplier := [5]float64{0, .45, .50, .55, .60}[rank]
	manaCost := [5]float64{0, 30, 40, 50, 65}[rank]
	level := [5]int{0, 16, 30, 44, 58}[rank]

	return core.SpellConfig{
		SpellCode:     SpellCode_HunterMongooseBite,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
		Flags:         core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagStrike,
		Rank:          rank,
		RequiredLevel: level,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: time.Second*5 - time.Millisecond*500*time.Duration(hunter.Talents.ViciousStrikes),
			},
		},

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget <= core.MaxMeleeAttackDistance
		},

		BonusCritRating:  hunter.savageStrikesCritRating(),
		CritDamageBonus:  hunter.killerInstinctCritDamageBonus(),
		DamageMultiplier: 1 + 0.05*float64(hunter.Talents.ViciousStrikes),
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			mhDamage := baseDamage + weaponMultiplier*spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
			result := spell.CalcAndDealDamage(sim, target, mhDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if hunter.MongooseBiteOH != nil && result.Landed() {
				hunter.MongooseBiteOH.Cast(sim, target)
			}
		},
	}
}

func (hunter *Hunter) registerMongooseBiteSpell() {
	rank := 0
	for r := 4; r >= 1; r-- {
		if hunter.getMongooseBiteConfig(r).RequiredLevel <= int(hunter.Level) {
			rank = r
			break
		}
	}
	if rank == 0 {
		return
	}

	if hunter.AutoAttacks.IsDualWielding {
		baseDamage := [5]float64{0, 15, 25, 45, 70}[rank]
		weaponMultiplier := [5]float64{0, .45, .50, .55, .60}[rank]
		spellId := [5]int32{0, 1495, 14269, 14270, 14271}[rank]

		hunter.MongooseBiteOH = hunter.RegisterSpell(core.SpellConfig{
			SpellCode:   SpellCode_HunterMongooseBite,
			ActionID:    core.ActionID{SpellID: spellId}.WithTag(2),
			SpellSchool: core.SpellSchoolPhysical,
			DefenseType: core.DefenseTypeMelee,
			ProcMask:    core.ProcMaskMeleeOHSpecial,
			Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete | SpellFlagStrike,

			BonusCritRating:  hunter.savageStrikesCritRating(),
			CritDamageBonus:  hunter.killerInstinctCritDamageBonus(),
			DamageMultiplier: 1 + 0.05*float64(hunter.Talents.ViciousStrikes),
			ThreatMultiplier: 1,
			BonusCoefficient: 1,

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				ohDamage := baseDamage + weaponMultiplier*spell.Unit.OHWeaponDamage(sim, spell.MeleeAttackPower(target))
				spell.CalcAndDealDamage(sim, target, ohDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
			},
		})
	}

	hunter.MongooseBite = hunter.GetOrRegisterSpell(hunter.getMongooseBiteConfig(rank))
}
