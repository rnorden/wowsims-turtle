package hunter

import (
	"fmt"
	"time"

	"github.com/isfir/wowsims-turtle/sim/core"
)

func (hunter *Hunter) registerVolleySpell() {
	for rank := 3; rank >= 1; rank-- {
		config := hunter.getVolleyConfig(rank)

		if config.RequiredLevel <= int(hunter.Level) {
			hunter.Volley = hunter.GetOrRegisterSpell(config)
			break
		}
	}
}

// Turtle WoW Volley: a cast (reduced by Barrage) that places an area effect ticking every second for 6 sec.
func (hunter *Hunter) getVolleyConfig(rank int) core.SpellConfig {
	spellId := [4]int32{0, 1510, 14294, 14295}[rank]
	baseDamage := [4]float64{0, 50, 75, 100}[rank]
	manaCost := [4]float64{0, 210, 270, 330}[rank]
	level := [4]int{0, 40, 50, 58}[rank]

	castTime := time.Second*3 - []time.Duration{0, 1000, 1500, 2000}[hunter.Talents.Barrage]*time.Millisecond

	return core.SpellConfig{
		SpellCode:   SpellCode_HunterVolley,
		ActionID:    core.ActionID{SpellID: spellId},
		SpellSchool: core.SpellSchoolArcane,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       core.SpellFlagAPL,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: castTime,
			},
			IgnoreHaste: true,
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				hunter.Unit.AutoAttacks.CancelAutoSwing(sim)
			},
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: time.Second * 60,
			},
		},

		Dot: core.DotConfig{
			IsAOE: true,
			Aura: core.Aura{
				Label: fmt.Sprintf("Volley (Rank %d)", rank),
			},
			NumberOfTicks:    6,
			TickLength:       time.Second * 1,
			BonusCoefficient: .056,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				for _, aoeTarget := range sim.Encounter.TargetUnits {
					dot.CalcAndDealPeriodicSnapshotDamage(sim, aoeTarget, dot.OutcomeTick)
				}
			},
		},

		CritDamageBonus:  hunter.mortalShots(),
		DamageMultiplier: 1 + 0.05*float64(hunter.Talents.Barrage),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			hunter.Unit.AutoAttacks.EnableAutoSwing(sim)
			spell.AOEDot().Apply(sim)
		},
	}
}
