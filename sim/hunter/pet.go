package hunter

import (
	"fmt"
	"time"

	"github.com/isfir/wowsims-turtle/sim/core"
	"github.com/isfir/wowsims-turtle/sim/core/proto"
	"github.com/isfir/wowsims-turtle/sim/core/stats"
)

type HunterPet struct {
	core.Pet

	config PetConfig

	hunterOwner *Hunter

	specialAbility   *core.Spell
	secondaryAbility *core.Spell
	focusDump        *core.Spell

	uptimePercent float64
}

func (hunter *Hunter) NewHunterPet() *HunterPet {
	if hunter.Options.PetType == proto.Hunter_Options_PetNone {
		return nil
	}
	if hunter.Options.PetUptime <= 0 {
		return nil
	}
	petConfig := PetConfigs[hunter.Options.PetType]

	hunterPetBaseStats := stats.Stats{}

	baseMinDamage := 0.0
	baseMaxDamage := 0.0
	attackSpeed := 2.0

	switch hunter.Options.PetAttackSpeed {
	case proto.Hunter_Options_One:
		attackSpeed = 1.0
	case proto.Hunter_Options_OneTwo:
		attackSpeed = 1.2
	case proto.Hunter_Options_OneThree:
		attackSpeed = 1.3
	case proto.Hunter_Options_OneFour:
		attackSpeed = 1.4
	case proto.Hunter_Options_OneFive:
		attackSpeed = 1.5
	case proto.Hunter_Options_OneSix:
		attackSpeed = 1.6
	case proto.Hunter_Options_OneSeven:
		attackSpeed = 1.7
	case proto.Hunter_Options_Two:
		attackSpeed = 2
	case proto.Hunter_Options_TwoFour:
		attackSpeed = 2.4
	case proto.Hunter_Options_TwoFive:
		attackSpeed = 2.5
	}

	// Base weapon damage per second of swing time, calibrated against an in-game level 60 Raptor
	// (1.5 speed, 557 AP, Unleashed Fury 5/5). Its sheet showed 122-154 when happy and 97-124 when
	// content, so the sheet includes happiness (x1.25). Removing AP (557/14*1.5) and the x1.32 Unleashed
	// Fury * Raptor modifiers leaves ~14-34 per swing at 1.5 speed. This assumes the client's Raptor
	// +10% family passive (17217) applies on Turtle.
	baseMinDamage = 9.35 * attackSpeed
	baseMaxDamage = 22.65 * attackSpeed

	// Level 60 base stats, from the same in-game pet. Str/Agi/Int/Spi matched vanilla; Stamina is what
	// remained after Endurance Training's 30% of hunter Stamina; base AP showed 260 (= 2*Str - 12) and armor
	// 4128 (= (3808 + 2 per Agility) * the Raptor's 1.03).
	hunterPetBaseStats = stats.Stats{
		stats.Strength:  136,
		stats.Agility:   100,
		stats.Stamina:   256,
		stats.Intellect: 50,
		stats.Spirit:    80,
		stats.Armor:     3808,

		stats.AttackPower: -12,
	}

	hp := &HunterPet{
		Pet:         core.NewPet(petConfig.Name, &hunter.Character, hunterPetBaseStats, hunter.makeStatInheritance(), true, false),
		config:      petConfig,
		hunterOwner: hunter,
	}

	hp.Pet.MobType = petConfig.MobType

	hp.EnableAutoAttacks(hp, core.AutoAttackOptions{
		MainHand: core.Weapon{
			BaseDamageMin: baseMinDamage,
			BaseDamageMax: baseMaxDamage,
			SwingSpeed:    attackSpeed,
		},
		AutoSwingMelee: true,
	})

	// Happiness
	hp.PseudoStats.DamageDealtMultiplier *= 1.25

	// Family scalars
	hp.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= hp.config.Damage
	hp.PseudoStats.ArmorMultiplier *= hp.config.Armor
	hp.MultiplyStat(stats.Health, hp.config.Health)

	hp.AddStatDependency(stats.Strength, stats.AttackPower, 2)

	// Warrior crit scaling
	hp.AddStatDependency(stats.Agility, stats.MeleeCrit, core.CritPerAgiAtLevel[proto.Class_ClassWarrior]*core.CritRatingPerCritChance)
	hp.AddStatDependency(stats.Intellect, stats.SpellCrit, core.CritPerIntAtLevel[proto.Class_ClassWarrior]*core.SpellCritRatingPerCritChance)

	core.ApplyPetConsumeEffects(&hp.Character, hunter.Consumes)

	hunter.AddPet(hp)

	return hp
}

func (hp *HunterPet) GetPet() *core.Pet {
	return &hp.Pet
}

func (hp *HunterPet) Initialize() {
	hp.specialAbility = hp.NewPetAbility(hp.config.SpecialAbility, true)
	hp.secondaryAbility = hp.NewPetAbility(hp.config.SecondaryAbility, false)
	hp.focusDump = hp.NewPetAbility(hp.config.FocusDump, false)

	hp.EnableFocusBar(1, func(sim *core.Simulation) {
		if hp.GCD.IsReady(sim) {
			hp.OnGCDReady(sim)
		}
	})
}

func (hp *HunterPet) Reset(_ *core.Simulation) {
	hp.uptimePercent = min(1, max(0, hp.hunterOwner.Options.PetUptime))
}

func (hp *HunterPet) ExecuteCustomRotation(sim *core.Simulation) {
	percentRemaining := sim.GetRemainingDurationPercent()
	if percentRemaining < 1.0-hp.uptimePercent { // once fight is % completed, disable pet.
		hp.Disable(sim)
		return
	}

	target := hp.CurrentTarget

	// using Cast() directly is very expensive, since cast failures are logged, involving string operations
	tryCast := func(spell *core.Spell) bool {
		if !spell.CanCast(sim, target) {
			return false
		}
		if !spell.Cast(sim, target) {
			panic(fmt.Sprintf("Cast failed after CanCast() for spell %d", spell.SpellID))
		}
		return true
	}

	if hp.focusDump == nil {
		if !tryCast(hp.specialAbility) && hp.GCD.IsReady(sim) {
			hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
		}
		return
	}
	if hp.specialAbility == nil {
		if !tryCast(hp.focusDump) && hp.GCD.IsReady(sim) {
			hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
		}
		return
	}

	if hp.config.CustomRotation != nil {
		hp.config.CustomRotation(sim, hp, tryCast)
	} else {
		// Priority: special ability, then the secondary ability, then the focus dump. An ability that is
		// off cooldown but unaffordable makes the pet pool focus for it.
		if hp.specialAbility.IsReady(sim) {
			if !tryCast(hp.specialAbility) && hp.GCD.IsReady(sim) {
				hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
			}
		} else if hp.secondaryAbility != nil && hp.secondaryAbility.IsReady(sim) {
			if !tryCast(hp.secondaryAbility) && hp.GCD.IsReady(sim) {
				hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
			}
		} else if hp.focusDump.IsReady(sim) {
			if !tryCast(hp.focusDump) && hp.GCD.IsReady(sim) {
				hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
			}
		}
	}
}

func (hunter *Hunter) makeStatInheritance() core.PetStatInheritance {
	// Spirit Bond: the pet gains melee attack power equal to 12.5% per rank of the hunter's ranged
	// attack power, and spell power equal to 7.5% per rank (rank values 12/25 and 7/15 in client data).
	apScale := []float64{0, 0.12, 0.25}[hunter.Talents.SpiritBond]
	spScale := []float64{0, 0.07, 0.15}[hunter.Talents.SpiritBond]

	// Endurance Training: the pet gains 6% per rank of the hunter's Stamina (client spell 19587).
	staScale := 0.06 * float64(hunter.Talents.EnduranceTraining)

	return func(ownerStats stats.Stats) stats.Stats {
		return stats.Stats{
			stats.AttackPower: ownerStats[stats.RangedAttackPower] * apScale,
			stats.SpellPower:  ownerStats[stats.RangedAttackPower] * spScale,
			stats.Stamina:     ownerStats[stats.Stamina] * staScale,
		}
	}
}

type PetConfig struct {
	Name    string
	MobType proto.MobType

	SpecialAbility   PetAbilityType
	SecondaryAbility PetAbilityType // optional, used between the special ability and the focus dump
	FocusDump        PetAbilityType

	Health float64
	Armor  float64
	Damage float64

	CustomRotation func(*core.Simulation, *HunterPet, func(*core.Spell) bool)
}

// Family Health/Armor/Damage modifiers match the client "Tamed Pet Passive (DND)" spells (e.g. 17217 Raptor:
// +10% damage, +3% armor, -5% health; 46242 Serpent).
// Abilities reference: https://wotlk.wowhead.com/hunter-pets
// https://wotlk.wowhead.com/guides/hunter-dps-best-pets-taming-loyalty-burning-crusade-classic
var PetConfigs = map[proto.Hunter_Options_PetType]PetConfig{
	proto.Hunter_Options_Cat: {
		Name:    "Cat",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite,
		FocusDump:      Claw,

		Health: 0.98,
		Armor:  1.00,
		Damage: 1.10,
	},
	proto.Hunter_Options_WindSerpent: {
		Name:    "Wind Serpent",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite,
		FocusDump:      LightningBreath,

		Health: 1.00,
		Armor:  1.00,
		Damage: 1.07,
	},
	proto.Hunter_Options_Bat: {
		Name:    "Bat",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite,
		FocusDump:      Screech,

		Health: 1.00,
		Armor:  1.00,
		Damage: 1.07,

		CustomRotation: func(sim *core.Simulation, hp *HunterPet, tryCast func(*core.Spell) bool) {
			if hp.specialAbility.CD.IsReady(sim) {
				if !tryCast(hp.specialAbility) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			} else {
				if !tryCast(hp.focusDump) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			}
		},
	},
	proto.Hunter_Options_Bear: {
		Name:    "Bear",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility:   RoarOfFortitude,
		SecondaryAbility: Bite,
		FocusDump:        Claw,

		Health: 1.08,
		Armor:  1.05,
		Damage: 0.91,
	},
	proto.Hunter_Options_Boar: {
		Name:    "Boar",
		MobType: proto.MobType_MobTypeBeast,

		//SpecialAbility: Gore,
		FocusDump: Bite,

		Health: 1.04,
		Armor:  1.09,
		Damage: 0.90,
	},
	proto.Hunter_Options_CarrionBird: {
		Name:    "Carrion Bird",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility:   Bite,
		SecondaryAbility: Screech,
		FocusDump:        Claw,

		Health: 1.00,
		Armor:  1.05,
		Damage: 1.00,
	},
	proto.Hunter_Options_Owl: {
		Name:    "Owl",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Screech,
		FocusDump:      Claw,

		Health: 1.00,
		Armor:  1.00,
		Damage: 1.07,
	},
	proto.Hunter_Options_Crab: {
		Name:    "Crab",
		MobType: proto.MobType_MobTypeBeast,

		FocusDump: Claw,

		Health: 0.96,
		Armor:  1.13,
		Damage: 0.95,
	},
	proto.Hunter_Options_Crocolisk: {
		Name:    "Crocolisk",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: DeathRoll,
		FocusDump:      Bite,

		Health: 0.95,
		Armor:  1.10,
		Damage: 1.00,
	},
	proto.Hunter_Options_Gorilla: {
		Name:    "Gorilla",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Thunderstomp,
		FocusDump:      Bite,

		Health: 1.04,
		Armor:  1.00,
		Damage: 1.02,
	},
	proto.Hunter_Options_Hyena: {
		Name:    "Hyena",
		MobType: proto.MobType_MobTypeBeast,

		FocusDump: Bite,

		Health: 1.00,
		Armor:  1.05,
		Damage: 1.00,
	},
	proto.Hunter_Options_Raptor: {
		Name:    "Raptor",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility:   Bite,
		SecondaryAbility: SavageRend,
		FocusDump:        Claw,

		Health: 0.95,
		Armor:  1.03,
		Damage: 1.10,
	},
	proto.Hunter_Options_Scorpid: {
		Name:    "Scorpid",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: ScorpidPoison,
		FocusDump:      Claw,

		Health: 1.00,
		Armor:  1.10,
		Damage: 0.94,

		CustomRotation: func(sim *core.Simulation, hp *HunterPet, tryCast func(*core.Spell) bool) {
			target := hp.CurrentTarget

			if (hp.specialAbility.Dot(target).GetStacks() < hp.specialAbility.Dot(target).MaxStacks || hp.specialAbility.Dot(target).RemainingDuration(sim) < time.Second*3) && hp.CurrentFocus() < 90 {
				if !tryCast(hp.specialAbility) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			} else {
				if !tryCast(hp.focusDump) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			}
		},
	},
	proto.Hunter_Options_Spider: {
		Name:    "Spider",
		MobType: proto.MobType_MobTypeBeast,

		FocusDump: Bite,

		Health: 1.00,
		Armor:  1.00,
		Damage: 1.07,
	},
	proto.Hunter_Options_Tallstrider: {
		Name:    "Tallstrider",
		MobType: proto.MobType_MobTypeBeast,

		FocusDump: Bite,

		Health: 1.05,
		Armor:  1.00,
		Damage: 1.00,
	},
	proto.Hunter_Options_Turtle: {
		Name:    "Turtle",
		MobType: proto.MobType_MobTypeBeast,

		// SpecialAbility: ShellShield,
		FocusDump: Bite,

		Health: 1.00,
		Armor:  1.13,
		Damage: 0.90,
	},
	proto.Hunter_Options_Wolf: {
		Name:    "Wolf",
		MobType: proto.MobType_MobTypeBeast,

		// Bite first: with only the hunter and pet in the party, Howl-first loses ~20 DPS. Howl's value
		// grows with each physical-damage party member, which a raid sim credits to them.
		SpecialAbility: Bite,
		FocusDump:      FuriousHowl,

		Health: 1.00,
		Armor:  1.05,
		Damage: 1.00,
	},
	// Turtle family (1.17.2). Learns Poison Spit and Bite; family stat modifiers are unknown.
	proto.Hunter_Options_Serpent: {
		Name:    "Serpent",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: PoisonSpit,
		FocusDump:      Bite,

		// Client spell 46242 "Tamed Pet Passive (DND) Serpent".
		Health: 1.03,
		Armor:  1.01,
		Damage: 1.06,
	},
}
