package hunter

import (
	"time"

	"github.com/isfir/wowsims-turtle/sim/core"
	"github.com/isfir/wowsims-turtle/sim/core/stats"
)

// Pet abilities added or reworked by Turtle WoW. Values are rank-60 client Spell.csv data
// (damage = basePoints+1 .. basePoints+dieSides), cross-checked against the "Book of Pets"
// sheet and the OctoWoW Turtle hunter changelog (t=2322).

// Furious Howl (Wolf), rank 4 (24597): party members within 40 yd deal 45-57 extra damage on their
// next physical attack, +1 per 25 pet AP. 50 focus, 10 sec cooldown, lasts 10 sec.
func (hp *HunterPet) newFuriousHowl() *core.Spell {
	actionID := core.ActionID{SpellID: 24597}

	var bonus float64
	var howlAuras []*core.Aura
	for _, agent := range hp.hunterOwner.Party.PlayersAndPets {
		unit := &agent.GetCharacter().Unit
		var applied float64
		howlAuras = append(howlAuras, unit.RegisterAura(core.Aura{
			Label:    "Furious Howl",
			ActionID: actionID,
			Duration: time.Second * 10,
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				applied = bonus
				aura.Unit.PseudoStats.BonusPhysicalDamage += applied
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				aura.Unit.PseudoStats.BonusPhysicalDamage -= applied
			},
			OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
				if spell.SpellSchool == core.SpellSchoolPhysical && spell.ProcMask.Matches(core.ProcMaskMeleeOrRanged) && result.Landed() {
					aura.Deactivate(sim)
				}
			},
		}))
	}

	return hp.RegisterSpell(core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete,

		FocusCost: core.FocusCostOptions{
			Cost: 50,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: hp.specialAbilityCooldown(time.Second * 10),
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			bonus = sim.Roll(45, 57) + hp.GetStat(stats.AttackPower)/25
			for _, aura := range howlAuras {
				aura.Deactivate(sim)
				aura.Activate(sim)
			}
		},
	})
}

// Poison Spit (Serpent), rank 3 (46273): 35 Nature damage every 2 sec for 8 sec,
// +0.25 spell power per tick. 35 focus, 8 sec cooldown.
func (hp *HunterPet) newPoisonSpit() *core.Spell {
	return hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 46273},
		SpellSchool: core.SpellSchoolNature,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       core.SpellFlagPoison,

		FocusCost: core.FocusCostOptions{
			Cost: 35,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: hp.specialAbilityCooldown(time.Second * 8),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Poison Spit",
			},
			NumberOfTicks:    4,
			TickLength:       time.Second * 2,
			BonusCoefficient: 0.25,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, 35, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				spell.Dot(target).Apply(sim)
			}
		},
	})
}

// Savage Rend (Raptor), rank 6 (36541): 27-36 physical damage plus a bleed of 16 per second for
// 4 sec; a critical strike extends the bleed by 1 sec. 35 focus, 10 sec cooldown.
func (hp *HunterPet) newSavageRend() *core.Spell {
	bleed := hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 36541}.WithTag(1),
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Savage Rend",
			},
			NumberOfTicks: 4,
			TickLength:    time.Second,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, 16, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},
	})

	return hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 36541},
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
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: hp.specialAbilityCooldown(time.Second * 10),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealDamage(sim, target, sim.Roll(27, 36), spell.OutcomeMeleeSpecialHitAndCrit)
			if !result.Landed() {
				return
			}
			dot := bleed.Dot(target)
			dot.NumberOfTicks = core.TernaryInt32(result.DidCrit(), 5, 4)
			dot.Apply(sim)
		},
	})
}

// Death Roll (Crocolisk), rank 6 (36553 -> 36547): 60 physical damage every 0.5 sec for 2.5 sec,
// 25% more against targets below 35% health. 50 focus, 30 sec cooldown.
func (hp *HunterPet) newDeathRoll() *core.Spell {
	hit := hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 36547},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			damage := 60.0
			if sim.IsExecutePhase35() {
				damage *= 1.25
			}
			spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMeleeSpecialHitAndCrit)
		},
	})

	return hp.RegisterSpell(core.SpellConfig{
		ActionID: core.ActionID{SpellID: 36553},
		Flags:    core.SpellFlagNoOnCastComplete,

		FocusCost: core.FocusCostOptions{
			Cost: 50,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: hp.specialAbilityCooldown(time.Second * 30),
			},
		},

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Death Roll",
			},
			NumberOfTicks: 5,
			TickLength:    time.Millisecond * 500,
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				hit.Cast(sim, target)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.Dot(target).Apply(sim)
		},
	})
}

// Roar of Fortitude (Bear), 36535: party members deal 3% more damage and take 8% less for 12 sec.
// 50 focus, 2 min cooldown.
func (hp *HunterPet) newRoarOfFortitude() *core.Spell {
	actionID := core.ActionID{SpellID: 36535}

	var roarAuras []*core.Aura
	for _, agent := range hp.hunterOwner.Party.PlayersAndPets {
		unit := &agent.GetCharacter().Unit
		roarAuras = append(roarAuras, unit.RegisterAura(core.Aura{
			Label:    "Roar of Fortitude",
			ActionID: actionID,
			Duration: time.Second * 12,
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				aura.Unit.PseudoStats.DamageDealtMultiplier *= 1.03
				aura.Unit.PseudoStats.DamageTakenMultiplier *= 0.92
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				aura.Unit.PseudoStats.DamageDealtMultiplier /= 1.03
				aura.Unit.PseudoStats.DamageTakenMultiplier /= 0.92
			},
		}))
	}

	return hp.RegisterSpell(core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete,

		FocusCost: core.FocusCostOptions{
			Cost: 50,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: hp.specialAbilityCooldown(time.Minute * 2),
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for _, aura := range roarAuras {
				aura.Activate(sim)
			}
		},
	})
}

// Thunderstomp (Gorilla), rank 4 (51156): 153-178 Nature damage to all nearby enemies,
// +0.21 spell power. 60 focus, 40 sec cooldown.
func (hp *HunterPet) newThunderstomp() *core.Spell {
	return hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 51156},
		SpellSchool: core.SpellSchoolNature,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,

		FocusCost: core.FocusCostOptions{
			Cost: 60,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: hp.specialAbilityCooldown(time.Second * 40),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1.5,
		BonusCoefficient: 0.21,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for _, aoeTarget := range sim.Encounter.TargetUnits {
				spell.CalcAndDealDamage(sim, aoeTarget, sim.Roll(153, 178), spell.OutcomeMagicHitAndCrit)
			}
		},
	})
}
