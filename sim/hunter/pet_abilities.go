package hunter

import (
	"time"

	"github.com/isfir/wowsims-turtle/sim/core"
)

type PetAbilityType int

// Pet AI doesn't use abilities immediately, so model this with a 1.6s GCD.
const PetGCD = time.Millisecond * 1600

const (
	Unknown PetAbilityType = iota
	Bite
	Claw
	Screech
	FuriousHowl
	LightningBreath
	ScorpidPoison
)

// Bestial Discipline (Turtle 2025-07-25 rework): reduces the cooldown of pet special abilities
// by 10/20% (client spell 19592 effect 2: -20% SPELLMOD_COOLDOWN).
func (hp *HunterPet) specialAbilityCooldown(base time.Duration) time.Duration {
	return time.Duration(float64(base) * (1 - 0.1*float64(hp.hunterOwner.Talents.BestialDiscipline)))
}

func (hp *HunterPet) NewPetAbility(abilityType PetAbilityType, isPrimary bool) *core.Spell {
	switch abilityType {
	case Bite:
		return hp.newBite()
	case Claw:
		return hp.newClaw()
	case Screech:
		return hp.newScreech()
	// case FuriousHowl:
	// 	return hp.newFuriousHowl()
	case LightningBreath:
		return hp.newLightningBreath()
	case ScorpidPoison:
		return hp.newScorpidPoison()
	// case Swipe:
	// 	return hp.newSwipe()
	case Unknown:
		return nil
	default:
		panic("Invalid pet ability type")
	}
}

func (hp *HunterPet) newClaw() *core.Spell {
	baseDamageMin := map[int32]float64{
		25: 16,
		40: 26,
		50: 35,
		60: 43,
	}[hp.Owner.Level]

	baseDamageMax := map[int32]float64{
		25: 22,
		40: 36,
		50: 49,
		60: 59,
	}[hp.Owner.Level]

	spellID := map[int32]int32{
		25: 16830,
		40: 16832,
		50: 3010,
		60: 3009,
	}[hp.Owner.Level]

	return hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellCode:   SpellCode_HunterPetClaw,
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics,

		FocusCost: core.FocusCostOptions{
			Cost: 25,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// Turtle: +1 damage per 42 pet attack power.
			baseDamage := sim.Roll(baseDamageMin, baseDamageMax) + spell.MeleeAttackPower(target)/42
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
		},
	})
}

func (hp *HunterPet) newBite() *core.Spell {
	baseDamageMin := map[int32]float64{
		25: 31,
		40: 49,
		50: 66,
		60: 81,
	}[hp.Owner.Level]

	baseDamageMax := map[int32]float64{
		25: 37,
		40: 59,
		50: 80,
		60: 99,
	}[hp.Owner.Level]

	spellID := map[int32]int32{
		25: 17257,
		40: 17259,
		50: 17260,
		60: 17261,
	}[hp.Owner.Level]

	return hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellCode:   SpellCode_HunterPetBite,
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics,

		FocusCost: core.FocusCostOptions{
			Cost: 35,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: hp.specialAbilityCooldown(6 * time.Second), // Turtle: 10s -> 6s
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// Turtle: +1 damage per 20 pet attack power.
			baseDamage := sim.Roll(baseDamageMin, baseDamageMax) + spell.MeleeAttackPower(target)/20
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
		},
	})
}

func (hp *HunterPet) newLightningBreath() *core.Spell {
	baseDamageMin := map[int32]float64{
		25: 36,
		40: 36,
		50: 78,
		60: 99,
	}[hp.Owner.Level]

	baseDamageMax := map[int32]float64{
		25: 41,
		40: 41,
		50: 91,
		60: 113,
	}[hp.Owner.Level]

	spellID := map[int32]int32{
		25: 25009,
		40: 25009, // rank 4 not available in SoD Phase 2
		50: 25011,
		60: 25012,
	}[hp.Owner.Level]

	return hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellCode:   SpellCode_HunterPetLightningBreath,
		SpellSchool: core.SpellSchoolNature,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,

		FocusCost: core.FocusCostOptions{
			Cost: 50,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 0.5, // Turtle: 0.5 spell power (pet spell power comes from Spirit Bond)

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageMin, baseDamageMax)

			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
		},
	})
}

func (hp *HunterPet) newScreech() *core.Spell {
	baseDamageMin := map[int32]float64{
		25: 12,
		40: 12,
		50: 19,
		60: 26,
	}[hp.Owner.Level]

	baseDamageMax := map[int32]float64{
		25: 16,
		40: 16,
		50: 25,
		60: 46,
	}[hp.Owner.Level]

	spellID := map[int32]int32{
		25: 24577,
		40: 24577,
		50: 24578,
		60: 24579,
	}[hp.Owner.Level]

	return hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellCode:   SpellCode_HunterPetScreech,
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeSpecial,
		Flags:       core.SpellFlagMeleeMetrics,

		FocusCost: core.FocusCostOptions{
			Cost: 20,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: hp.specialAbilityCooldown(5 * time.Second), // Turtle: 5s cooldown
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// Turtle: +1 damage per 25 pet attack power.
			baseDamage := sim.Roll(baseDamageMin, baseDamageMax) + spell.MeleeAttackPower(target)/25
			// This ability also applies a melee attack power reduction similar to demoralizing shout - left it out for now
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
		},
	})
}

// func (hp *HunterPet) newFuriousHowl() *core.Spell {
// 	actionID := core.ActionID{SpellID: 64495}

// 	petAura := hp.NewTemporaryStatsAura("FuriousHowl", actionID, stats.Stats{stats.AttackPower: 320, stats.RangedAttackPower: 320}, time.Second*20)
// 	ownerAura := hp.hunterOwner.NewTemporaryStatsAura("FuriousHowl", actionID, stats.Stats{stats.AttackPower: 320, stats.RangedAttackPower: 320}, time.Second*20)

// 	howlSpell := hp.RegisterSpell(core.SpellConfig{
// 		ActionID: actionID,

// 		FocusCost: core.FocusCostOptions{
// 			Cost: 20,
// 		},
// 		Cast: core.CastConfig{
// 			CD: core.Cooldown{
// 				Timer:    hp.NewTimer(),
// 				Duration: time.Second * 40,
// 			},
// 		},
// 		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
// 			return hp.IsEnabled()
// 		},
// 		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
// 			petAura.Activate(sim)
// 			ownerAura.Activate(sim)
// 		},
// 	})

// 	hp.hunterOwner.RegisterSpell(core.SpellConfig{
// 		ActionID: actionID,
// 		Flags:    core.SpellFlagAPL | core.SpellFlagMCD,
// 		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
// 			return howlSpell.CanCast(sim, target)
// 		},
// 		ApplyEffects: func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
// 			howlSpell.Cast(sim, target)
// 		},
// 	})

// 	hp.hunterOwner.AddMajorCooldown(core.MajorCooldown{
// 		Spell: howlSpell,
// 		Type:  core.CooldownTypeDPS,
// 	})

// 	return nil
// }

func (hp *HunterPet) newScorpidPoison() *core.Spell {
	// Client tick values (bp+1); Turtle raised base damage by 20% server-side (2024-10-12).
	baseDamageTick := 1.2 * map[int32]float64{
		25: 3,
		40: 6,
		50: 6,
		60: 8,
	}[hp.Owner.Level]
	spellID := map[int32]int32{
		25: 24583,
		40: 24586,
		50: 24586,
		60: 24587,
	}[hp.Owner.Level]

	return hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellCode:   SpellCode_HunterPetScorpidPoison,
		SpellSchool: core.SpellSchoolNature,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagPassiveSpell | core.SpellFlagPoison,

		FocusCost: core.FocusCostOptions{
			Cost: 25, // Turtle: 30 -> 25
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: hp.specialAbilityCooldown(time.Second * 3), // Turtle: 4s -> 3s
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label:     "ScorpidPoison",
				MaxStacks: 5,
				Duration:  time.Second * 10,
			},
			NumberOfTicks: 5,
			TickLength:    time.Second * 2,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, applyStack bool) {
				if !applyStack {
					return
				}

				// only the first stack snapshots the multiplier
				if dot.GetStacks() == 1 {
					attackTable := dot.Spell.Unit.AttackTables[target.UnitIndex][dot.Spell.CastType]
					dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(attackTable, true)
					dot.SnapshotBaseDamage = 0
				}

				// Turtle: 0.1 spell power per periodic tick, per stack.
				dot.SnapshotBaseDamage += baseDamageTick + 0.1*dot.Spell.GetBonusDamage(target)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMeleeSpecialHit)
			if !result.Landed() {
				return
			}

			dot := spell.Dot(target)
			dot.ApplyOrRefresh(sim)
			if dot.GetStacks() < dot.MaxStacks {
				dot.AddStack(sim)
				dot.TakeSnapshot(sim, true)
			}
		},
	})
}
