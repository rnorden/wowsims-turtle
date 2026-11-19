package hunter

import (
	"time"

	"github.com/isfir/wowsims-turtle/sim/core"
	"github.com/isfir/wowsims-turtle/sim/core/proto"
	"github.com/isfir/wowsims-turtle/sim/core/stats"
)

// Talent values follow the Turtle WoW client data (Spell.csv / Talent.dbc).
func (hunter *Hunter) ApplyTalents() {
	hunter.applyBeastMasteryTalents()
	hunter.applyMarksmanshipTalents()
	hunter.applySurvivalTalents()
}

// Beast Mastery

func (hunter *Hunter) applyBeastMasteryTalents() {
	if hunter.pet == nil {
		return
	}
	pet := hunter.pet

	pet.MultiplyStat(stats.Health, 1+0.02*float64(hunter.Talents.EnduranceTraining))
	pet.PseudoStats.DamageDealtMultiplier *= 1 + 0.04*float64(hunter.Talents.UnleashedFury)
	pet.AddStat(stats.MeleeCrit, 3*float64(hunter.Talents.Ferocity)*core.CritRatingPerCritChance)
	pet.AddStat(stats.SpellCrit, 3*float64(hunter.Talents.Ferocity)*core.SpellCritRatingPerCritChance)

	if hunter.Talents.BestialPrecision > 0 {
		pet.AddStat(stats.MeleeHit, 4*float64(hunter.Talents.BestialPrecision)*core.MeleeHitRatingPerHitChance)
		pet.AddStat(stats.SpellHit, 9*float64(hunter.Talents.BestialPrecision)*core.SpellHitRatingPerHitChance)
	}

	if hunter.Talents.BestialDiscipline > 0 {
		core.MakePermanent(hunter.RegisterAura(core.Aura{
			Label: "Bestial Discipline",
			OnInit: func(aura *core.Aura, sim *core.Simulation) {
				pet.AddFocusRegenMultiplier(0.1 * float64(hunter.Talents.BestialDiscipline))
			},
		}))
	}

	hunter.applyFrenzy()
	hunter.applyScentOfBlood()
	hunter.applyCoordinatedAssault()
}

// Frenzy: pet crits have a 20% chance per rank to grant 30% attack speed for 8 sec.
func (hunter *Hunter) applyFrenzy() {
	if hunter.Talents.Frenzy == 0 {
		return
	}

	procChance := 0.2 * float64(hunter.Talents.Frenzy)

	procAura := hunter.pet.RegisterAura(core.Aura{
		Label:    "Frenzy Proc",
		ActionID: core.ActionID{SpellID: 19615},
		Duration: time.Second * 8,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyAttackSpeed(sim, 1.3)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyAttackSpeed(sim, 1/1.3)
		},
	})

	core.MakePermanent(hunter.pet.RegisterAura(core.Aura{
		Label: "Frenzy",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.DidCrit() && sim.Proc(procChance, "Frenzy") {
				procAura.Activate(sim)
			}
		},
	}))
}

// Scent of Blood (52995) increases the pet's damage by 40%. The Scent of Blood talent grants it for
// 8 sec on 5% per rank of the hunter's attacks; Bestial Wrath grants it for 18 sec.
func (hunter *Hunter) registerScentOfBloodPetAura() *core.Aura {
	if hunter.ScentOfBloodPetAura == nil {
		hunter.ScentOfBloodPetAura = hunter.pet.RegisterAura(core.Aura{
			Label:    "Scent of Blood",
			ActionID: core.ActionID{SpellID: 52995},
			Duration: time.Second * 8,
		}).AttachMultiplicativePseudoStatBuff(&hunter.pet.PseudoStats.DamageDealtMultiplier, 1.4)
	}
	return hunter.ScentOfBloodPetAura
}

func (hunter *Hunter) activateScentOfBlood(sim *core.Simulation, duration time.Duration) {
	aura := hunter.ScentOfBloodPetAura
	if !hunter.pet.IsEnabled() {
		return
	}
	if aura.IsActive() && aura.RemainingDuration(sim) >= duration {
		return
	}
	aura.Duration = duration
	aura.Activate(sim)
	aura.Refresh(sim)
}

func (hunter *Hunter) applyScentOfBlood() {
	if hunter.Talents.ScentOfBlood > 0 {
		hunter.registerScentOfBloodPetAura()
		procChance := 0.05 * float64(hunter.Talents.ScentOfBlood)

		core.MakePermanent(hunter.RegisterAura(core.Aura{
			Label: "Scent of Blood Talent",
			OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
				if !spell.ProcMask.Matches(core.ProcMaskMeleeOrRanged) || !result.Landed() {
					return
				}
				if sim.Proc(procChance, "Scent of Blood") {
					hunter.activateScentOfBlood(sim, time.Second*8)
				}
			},
		}))
	}

	if !hunter.Talents.BestialWrath {
		return
	}
	hunter.registerScentOfBloodPetAura()

	bwSpell := hunter.RegisterSpell(core.SpellConfig{
		ActionID: core.ActionID{SpellID: 19574},
		Flags:    core.SpellFlagAPL | core.SpellFlagMCD,

		ManaCost: core.ManaCostOptions{
			BaseCost: 0.12,
		},
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: time.Second * 90,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.pet.IsEnabled()
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			hunter.activateScentOfBlood(sim, time.Second*18)
		},
	})

	hunter.AddMajorCooldown(core.MajorCooldown{
		Spell: bwSpell,
		Type:  core.CooldownTypeDPS,
	})
}

// Coordinated Assault: Arcane Shot, Steady Shot and Raptor Strike hits make the pet's next attack
// within 6 sec also deal Physical damage equal to 20% of the hunter's attack power. 3 sec cooldown.
func (hunter *Hunter) applyCoordinatedAssault() {
	if !hunter.Talents.CoordinatedAssault {
		return
	}

	pet := hunter.pet
	bonusDamage := 0.0

	strikeTogether := pet.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 49557},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		// The tooltip specifies a flat 20% of the hunter's attack power, so pet damage
		// modifiers (happiness, Unleashed Fury, Scent of Blood) are not applied.
		Flags: core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell | core.SpellFlagIgnoreAttackerModifiers,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, bonusDamage, spell.OutcomeMeleeSpecialCritOnly)
		},
	})

	petAura := pet.RegisterAura(core.Aura{
		Label:    "Strike Together",
		ActionID: core.ActionID{SpellID: 49558},
		Duration: time.Second * 6,
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell == strikeTogether || !spell.ProcMask.Matches(core.ProcMaskMelee) || !result.Landed() {
				return
			}
			aura.Deactivate(sim)
			strikeTogether.Cast(sim, result.Target)
		},
	})

	icd := core.Cooldown{
		Timer:    hunter.NewTimer(),
		Duration: time.Second * 3,
	}

	core.MakePermanent(hunter.RegisterAura(core.Aura{
		Label: "Coordinated Assault",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			switch spell.SpellCode {
			case SpellCode_HunterArcaneShot, SpellCode_HunterSteadyShot, SpellCode_HunterRaptorStrikeHit:
			default:
				return
			}
			if !result.Landed() || !icd.IsReady(sim) || !pet.IsEnabled() {
				return
			}
			if spell.SpellCode == SpellCode_HunterRaptorStrikeHit {
				bonusDamage = 0.2 * spell.MeleeAttackPower(result.Target)
			} else {
				bonusDamage = 0.2 * spell.RangedAttackPower(result.Target, false)
			}
			icd.Use(sim)
			petAura.Activate(sim)
		},
	}))
}

// Kill Command: usable after the hunter crits; the pet instantly attacks for 80% of its attack power.
func (hunter *Hunter) registerKillCommandSpell() {
	if hunter.KillCommandReadyAura == nil {
		return
	}

	pet := hunter.pet
	petStrike := pet.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_HunterKillCommand,
		ActionID:    core.ActionID{SpellID: 41828},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			damage := 0.8 * spell.MeleeAttackPower(target)
			spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMeleeSpecialHitAndCrit)
		},
	})

	hunter.KillCommand = hunter.RegisterSpell(core.SpellConfig{
		ActionID: core.ActionID{SpellID: 41827},
		Flags:    core.SpellFlagAPL,

		ManaCost: core.ManaCostOptions{
			BaseCost: 0.05,
		},
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: time.Second * 8,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.KillCommandReadyAura.IsActive() && pet.IsEnabled()
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
			hunter.KillCommandReadyAura.Deactivate(sim)
			petStrike.Cast(sim, target)
		},
	})
}

// Marksmanship

func (hunter *Hunter) mortalShots() float64 {
	return 0.06 * float64(hunter.Talents.MortalShots)
}

func (hunter *Hunter) applyMarksmanshipTalents() {
	// Efficiency: -2% mana cost per rank on Shots, Stings and Volley.
	if hunter.Talents.Efficiency > 0 {
		hunter.OnSpellRegistered(func(spell *core.Spell) {
			if spell.Cost != nil && (spell.Flags.Matches(SpellFlagSting|SpellFlagShot) || spell.SpellCode == SpellCode_HunterVolley) {
				spell.Cost.Multiplier -= 2 * hunter.Talents.Efficiency
			}
		})
	}

	// Lethal Shots: +1% ranged crit per rank.
	if hunter.Talents.LethalShots > 0 {
		lethalBonus := float64(hunter.Talents.LethalShots) * core.CritRatingPerCritChance
		hunter.OnSpellRegistered(func(spell *core.Spell) {
			if spell.Flags.Matches(SpellFlagShot) {
				spell.BonusCritRating += lethalBonus
			}
		})
		hunter.AutoAttacks.RangedConfig().BonusCritRating += lethalBonus
	}

	// Improved Stings: +6% Serpent Sting damage per rank.
	if hunter.Talents.ImprovedStings > 0 {
		hunter.OnSpellRegistered(func(spell *core.Spell) {
			if spell.SpellCode == SpellCode_HunterSerpentSting {
				spell.DamageMultiplier *= 1 + 0.06*float64(hunter.Talents.ImprovedStings)
			}
		})
	}

	// Ranged Weapon Specialization: +2% ranged weapon damage per rank.
	if hunter.Talents.RangedWeaponSpecialization > 0 {
		mult := 1 + 0.02*float64(hunter.Talents.RangedWeaponSpecialization)
		hunter.OnSpellRegistered(func(spell *core.Spell) {
			if spell.ProcMask.Matches(core.ProcMaskRanged) && spell.SpellCode != SpellCode_HunterSerpentSting {
				spell.DamageMultiplier *= mult
			}
		})
	}

	hunter.applyEndlessQuiver()
	hunter.applyPiercingShots()
	hunter.applyLockAndLoad()
}

// Endless Quiver: Auto Shot, Arcane Shot, Steady Shot and Multi-Shot have a 3% chance per rank
// to fire an additional normal ranged attack.
func (hunter *Hunter) applyEndlessQuiver() {
	if hunter.Talents.EndlessQuiver == 0 {
		return
	}

	procChance := 0.03 * float64(hunter.Talents.EndlessQuiver)

	core.MakePermanent(hunter.RegisterAura(core.Aura{
		Label: "Endless Quiver",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if hunter.ExtraShot == nil || spell.SpellCode == SpellCode_HunterExtraShot {
				return
			}
			isTrigger := spell == hunter.AutoAttacks.RangedAuto() ||
				spell.SpellCode == SpellCode_HunterArcaneShot ||
				spell.SpellCode == SpellCode_HunterSteadyShot ||
				spell.SpellCode == SpellCode_HunterMultiShot
			if !isTrigger {
				return
			}
			if sim.Proc(procChance, "Endless Quiver") {
				hunter.ExtraShot.Cast(sim, result.Target)
			}
		},
	}))
}

// Piercing Shots: Multi-Shot, Steady Shot and Aimed Shot crits make the target bleed for 15% per
// rank of the damage dealt over 8 sec. Unticked damage rolls into a refreshed bleed.
func (hunter *Hunter) applyPiercingShots() {
	if hunter.Talents.PiercingShots == 0 {
		return
	}

	bleedFraction := 0.15 * float64(hunter.Talents.PiercingShots)

	hunter.PiercingShots = hunter.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_HunterPiercingShots,
		ActionID:    core.ActionID{SpellID: 51514},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeRanged,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell | core.SpellFlagIgnoreAttackerModifiers,

		DamageMultiplier: 1,
		ThreatMultiplier: 0,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Piercing Shots",
			},
			NumberOfTicks: 4,
			TickLength:    time.Second * 2,

			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},
	})

	core.MakePermanent(hunter.RegisterAura(core.Aura{
		Label: "Piercing Shots Talent",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			switch spell.SpellCode {
			case SpellCode_HunterMultiShot, SpellCode_HunterSteadyShot, SpellCode_HunterAimedShot:
			default:
				return
			}
			if !result.DidCrit() {
				return
			}

			dot := hunter.PiercingShots.Dot(result.Target)
			remaining := 0.0
			if dot.IsActive() {
				remaining = dot.SnapshotBaseDamage * float64(dot.MaxTicksRemaining())
			}
			total := remaining + result.Damage*bleedFraction

			dot.SnapshotBaseDamage = total / float64(dot.OriginalNumberOfTicks)
			dot.SnapshotAttackerMultiplier = 1
			dot.Apply(sim)
		},
	}))
}

// Lock and Load: Steady Shot, Aimed Shot and Arcane Shot crits reset the cooldown of Aimed Shot and
// reduce the cast time of the next Aimed Shot within 10 sec by 1 sec.
func (hunter *Hunter) applyLockAndLoad() {
	if !hunter.Talents.LockAndLoad || !hunter.Talents.AimedShot {
		return
	}

	const castReduction = time.Second

	hunter.LockAndLoadAura = hunter.RegisterAura(core.Aura{
		Label:    "Lock and Load",
		ActionID: core.ActionID{SpellID: 52921},
		Duration: time.Second * 10,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			hunter.AimedShot.DefaultCast.CastTime -= castReduction
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			hunter.AimedShot.DefaultCast.CastTime += castReduction
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if spell.SpellCode == SpellCode_HunterAimedShot {
				aura.Deactivate(sim)
			}
		},
	})

	core.MakePermanent(hunter.RegisterAura(core.Aura{
		Label: "Lock and Load Talent",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			switch spell.SpellCode {
			case SpellCode_HunterSteadyShot, SpellCode_HunterAimedShot, SpellCode_HunterArcaneShot:
			default:
				return
			}
			if !result.DidCrit() || hunter.AimedShot == nil {
				return
			}
			hunter.AimedShot.CD.Reset()
			hunter.LockAndLoadAura.Activate(sim)
		},
	}))
}

// Survival

// Savage Strikes: +3% crit chance per rank for Lacerate, Raptor Strike, Mongoose Bite, Carve and Wing Clip.
func (hunter *Hunter) savageStrikesCritRating() float64 {
	return 3 * float64(hunter.Talents.SavageStrikes) * core.CritRatingPerCritChance
}

// Killer Instinct: increases the melee critical strike damage bonus.
func (hunter *Hunter) killerInstinctCritDamageBonus() float64 {
	return []float64{0, .07, .14, .20}[hunter.Talents.KillerInstinct]
}

func (hunter *Hunter) applySurvivalTalents() {
	// Improved Slaying: +1% damage and +1% critical damage per rank against Beasts, Giants, Dragonkin and Humanoids.
	if hunter.Talents.ImprovedSlaying > 0 {
		multiplier := 1 + 0.01*float64(hunter.Talents.ImprovedSlaying)
		hunter.Env.RegisterPostFinalizeEffect(func() {
			for _, t := range hunter.Env.Encounter.Targets {
				switch t.MobType {
				case proto.MobType_MobTypeBeast, proto.MobType_MobTypeGiant, proto.MobType_MobTypeDragonkin, proto.MobType_MobTypeHumanoid:
					for _, at := range hunter.AttackTables[t.UnitIndex] {
						at.DamageDealtMultiplier *= multiplier
						at.CritMultiplier *= multiplier
					}
				}
			}
		})
	}

	// Resourcefulness: -2% mana cost per rank on traps and melee abilities.
	if hunter.Talents.Resourcefulness > 0 {
		hunter.OnSpellRegistered(func(spell *core.Spell) {
			if spell.Cost != nil && spell.Flags.Matches(SpellFlagTrap|SpellFlagStrike) {
				spell.Cost.Multiplier -= 2 * hunter.Talents.Resourcefulness
			}
		})
	}

	// Swift Reflexes: +1% attack speed per rank (melee and ranged).
	if hunter.Talents.SwiftReflexes > 0 {
		multiplier := 1 + 0.01*float64(hunter.Talents.SwiftReflexes)
		hunter.PseudoStats.MeleeSpeedMultiplier *= multiplier
		hunter.PseudoStats.RangedSpeedMultiplier *= multiplier
	}

	// Savage Strikes: +12.5% off-hand damage per rank (crit part is in savageStrikesCritRating).
	if hunter.Talents.SavageStrikes > 0 {
		ohMultiplier := []float64{1, 1.13, 1.25}[hunter.Talents.SavageStrikes]
		hunter.AutoAttacks.OHConfig().DamageMultiplier *= ohMultiplier
		hunter.OnSpellRegistered(func(spell *core.Spell) {
			if spell.ProcMask.Matches(core.ProcMaskMeleeOHSpecial) {
				spell.DamageMultiplier *= ohMultiplier
			}
		})
	}

	// Alone Against the World: +3% damage per rank while the hunter has no pet.
	if hunter.Talents.AloneAgainstTheWorld > 0 && hunter.pet == nil {
		hunter.PseudoStats.DamageDealtMultiplier *= 1 + 0.03*float64(hunter.Talents.AloneAgainstTheWorld)
	}

	if hunter.Talents.Survivalist > 0 {
		hunter.MultiplyStat(stats.Health, 1+0.02*float64(hunter.Talents.Survivalist))
	}

	// Surefooted: +1% hit per rank, and another +1% per rank for melee while dual wielding.
	if hunter.Talents.Surefooted > 0 {
		hunter.AddStat(stats.MeleeHit, float64(hunter.Talents.Surefooted)*core.MeleeHitRatingPerHitChance)
		hunter.AddStat(stats.SpellHit, float64(hunter.Talents.Surefooted)*core.SpellHitRatingPerHitChance)

		if hunter.AutoAttacks.IsDualWielding {
			dwBonus := float64(hunter.Talents.Surefooted) * core.MeleeHitRatingPerHitChance
			hunter.AutoAttacks.MHConfig().BonusHitRating += dwBonus
			hunter.AutoAttacks.OHConfig().BonusHitRating += dwBonus
			hunter.OnSpellRegistered(func(spell *core.Spell) {
				if spell.ProcMask.Matches(core.ProcMaskMeleeSpecial) {
					spell.BonusHitRating += dwBonus
				}
			})
		}
	}

	// Killer Instinct: +1% crit per rank with all attacks; melee crit damage bonus via killerInstinctCritDamageBonus.
	if hunter.Talents.KillerInstinct > 0 {
		hunter.AddStat(stats.MeleeCrit, float64(hunter.Talents.KillerInstinct)*core.CritRatingPerCritChance)
		bonus := hunter.killerInstinctCritDamageBonus()
		hunter.AutoAttacks.MHConfig().CritDamageBonus += bonus
		hunter.AutoAttacks.OHConfig().CritDamageBonus += bonus
	}

	// Trap Mastery: +10% trap damage and -4/7/10% chance for trap effects to be resisted.
	if hunter.Talents.TrapMastery > 0 {
		hitBonus := []float64{0, 4, 7, 10}[hunter.Talents.TrapMastery]
		hunter.OnSpellRegistered(func(spell *core.Spell) {
			if spell.Flags.Matches(SpellFlagTrap) {
				spell.DamageMultiplier *= 1 + 0.1*float64(hunter.Talents.TrapMastery)
				spell.BonusHitRating += hitBonus
			}
		})
	}

	// Lightning Reflexes: +2% Agility per rank, and melee attack power equal to 20% of Agility per rank.
	if hunter.Talents.LightningReflexes > 0 {
		hunter.MultiplyStat(stats.Agility, 1+0.02*float64(hunter.Talents.LightningReflexes))
		hunter.AddStatDependency(stats.Agility, stats.AttackPower, 0.2*float64(hunter.Talents.LightningReflexes))
	}
}
