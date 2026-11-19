package hunter

import (
	"time"

	"github.com/isfir/wowsims-turtle/sim/core"
)

func (hunter *Hunter) registerRapidFire() {
	if hunter.Level < 26 {
		return
	}

	actionID := core.ActionID{SpellID: 3045}
	cooldown := time.Minute * 5

	hunter.RapidFireAura = hunter.RegisterAura(core.Aura{
		Label:    "Rapid Fire",
		ActionID: actionID,
		Duration: time.Second * 15,

		// Turtle WoW: +40% attack speed, and -40% cast time on Aimed Shot and Steady Shot.
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyAttackSpeed(sim, 1.4)
			for _, spell := range []*core.Spell{hunter.AimedShot, hunter.SteadyShot} {
				if spell != nil {
					spell.CastTimeMultiplier *= 0.6
				}
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyAttackSpeed(sim, 1/1.4)
			for _, spell := range []*core.Spell{hunter.AimedShot, hunter.SteadyShot} {
				if spell != nil {
					spell.CastTimeMultiplier /= 0.6
				}
			}
		},
	})

	hunter.RapidFire = hunter.RegisterSpell(core.SpellConfig{
		ActionID: actionID,

		ManaCost: core.ManaCostOptions{
			FlatCost: 100,
		},
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: cooldown,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			hunter.RapidFireAura.Activate(sim)
		},
	})

	hunter.AddMajorCooldown(core.MajorCooldown{
		Spell: hunter.RapidFire,
		Type:  core.CooldownTypeDPS,
	})
}
