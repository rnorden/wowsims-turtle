package hunter

import (
	"strconv"
	"time"

	"github.com/isfir/wowsims-turtle/sim/core"
	"github.com/isfir/wowsims-turtle/sim/core/stats"
)

// Swift Aspects: while an aspect is active, normal attacks of the matching type have a 10% chance
// to grant 3% attack speed per talent rank for 12 sec (Quick Shots for Hawk, Quick Strikes for Wolf).
const swiftAspectsProcChance = 0.1

func (hunter *Hunter) newSwiftAspectsAura(label string, spellID int32, ranged bool) *core.Aura {
	if hunter.Talents.SwiftAspects == 0 {
		return nil
	}

	multiplier := 1 + 0.03*float64(hunter.Talents.SwiftAspects)
	return hunter.GetOrRegisterAura(core.Aura{
		Label:    label,
		ActionID: core.ActionID{SpellID: spellID + hunter.Talents.SwiftAspects - 1},
		Duration: time.Second * 12,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			if ranged {
				aura.Unit.MultiplyRangedSpeed(sim, multiplier)
			} else {
				aura.Unit.MultiplyMeleeSpeed(sim, multiplier)
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			if ranged {
				aura.Unit.MultiplyRangedSpeed(sim, 1/multiplier)
			} else {
				aura.Unit.MultiplyMeleeSpeed(sim, 1/multiplier)
			}
		},
	})
}

type aspectConfig struct {
	name        string
	spellIDs    [8]int32
	levels      [8]int
	attackPower [8]float64
	manaCost    [8]float64
	stat        stats.Stat
	multiplier  *float64

	swiftLabel   string
	swiftSpellID int32
	swiftRanged  bool
	swiftMask    core.ProcMask
}

func (hunter *Hunter) registerAspectSpell(config aspectConfig) {
	rank := 0
	for i := 7; i > 0; i-- {
		if config.levels[i] <= int(hunter.Level) {
			rank = i
			break
		}
	}
	if rank == 0 {
		return
	}

	swiftAura := hunter.newSwiftAspectsAura(config.swiftLabel, config.swiftSpellID, config.swiftRanged)
	attackPower := config.attackPower[rank]
	actionID := core.ActionID{SpellID: config.spellIDs[rank]}

	aspectAura := hunter.GetOrRegisterAura(core.Aura{
		Label:    config.name + strconv.Itoa(rank),
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatDynamic(sim, config.stat, attackPower**config.multiplier)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatDynamic(sim, config.stat, -attackPower**config.multiplier)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if swiftAura == nil || !spell.ProcMask.Matches(config.swiftMask) || !result.Landed() {
				return
			}
			if sim.Proc(swiftAspectsProcChance, config.swiftLabel) {
				swiftAura.Activate(sim)
			}
		},
	})

	aspectAura.NewExclusiveEffect("Aspect", true, core.ExclusiveEffect{})

	hunter.GetOrRegisterSpell(core.SpellConfig{
		ActionID:      actionID,
		Flags:         core.SpellFlagAPL,
		Rank:          rank,
		RequiredLevel: config.levels[rank],

		ManaCost: core.ManaCostOptions{
			FlatCost: config.manaCost[rank],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return !aspectAura.IsActive()
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			aspectAura.Activate(sim)
		},
	})
}

func (hunter *Hunter) registerAspectOfTheHawkSpell() {
	hunter.AspectOfTheHawkAPMultiplier = 1.0
	hunter.registerAspectSpell(aspectConfig{
		name:        "Aspect of the Hawk",
		spellIDs:    [8]int32{0, 13165, 14318, 14319, 14320, 14321, 14322, 25296},
		levels:      [8]int{0, 10, 18, 28, 38, 48, 58, core.TernaryInt(core.IncludeAQ, 60, 61)},
		attackPower: [8]float64{0, 20, 35, 50, 70, 90, 110, 120},
		manaCost:    [8]float64{0, 20, 35, 50, 70, 90, 110, 120},
		stat:        stats.RangedAttackPower,
		multiplier:  &hunter.AspectOfTheHawkAPMultiplier,

		swiftLabel:   "Quick Shots",
		swiftSpellID: 51547,
		swiftRanged:  true,
		swiftMask:    core.ProcMaskRangedAuto,
	})
}

func (hunter *Hunter) registerAspectOfTheWolfSpell() {
	hunter.AspectOfTheWolfAPMultiplier = 1.0
	hunter.registerAspectSpell(aspectConfig{
		name:        "Aspect of the Wolf",
		spellIDs:    [8]int32{0, 45650, 51496, 51497, 51498, 51499, 51500, 51501},
		levels:      [8]int{0, 10, 18, 28, 38, 48, 58, 60},
		attackPower: [8]float64{0, 20, 35, 50, 70, 90, 110, 120},
		manaCost:    [8]float64{0, 20, 35, 50, 70, 90, 110, 120},
		stat:        stats.AttackPower,
		multiplier:  &hunter.AspectOfTheWolfAPMultiplier,

		swiftLabel:   "Quick Strikes",
		swiftSpellID: 51542,
		swiftRanged:  false,
		swiftMask:    core.ProcMaskMeleeWhiteHit,
	})
}

// Aspect of the Viper (45651, level 56): regenerates 5% of maximum mana every 5 sec.
// Shares the Aspect exclusivity with Hawk and Wolf, so swapping costs the hunter its attack power aspect.
func (hunter *Hunter) registerAspectOfTheViperSpell() {
	if hunter.Level < 56 {
		return
	}

	actionID := core.ActionID{SpellID: 45651}
	manaMetrics := hunter.NewManaMetrics(actionID)
	var pa *core.PendingAction

	viperAura := hunter.GetOrRegisterAura(core.Aura{
		Label:    "Aspect of the Viper",
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			pa = core.NewPeriodicAction(sim, core.PeriodicActionOptions{
				Period: time.Second * 5,
				OnAction: func(sim *core.Simulation) {
					hunter.AddMana(sim, 0.05*hunter.MaxMana(), manaMetrics)
				},
			})
			sim.AddPendingAction(pa)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			if pa != nil {
				pa.Cancel(sim)
				pa = nil
			}
		},
	})
	viperAura.NewExclusiveEffect("Aspect", true, core.ExclusiveEffect{})

	hunter.GetOrRegisterSpell(core.SpellConfig{
		ActionID:      actionID,
		Flags:         core.SpellFlagAPL,
		RequiredLevel: 56,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return !viperAura.IsActive()
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			viperAura.Activate(sim)
		},
	})
}
