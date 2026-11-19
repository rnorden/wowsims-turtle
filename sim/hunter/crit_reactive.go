package hunter

import (
	"time"

	"github.com/isfir/wowsims-turtle/sim/core"
)

// Kill Command and Lacerate can only be used after the hunter lands a critical strike.
// The client data does not specify how long the window lasts; MaNGOS-family servers clear
// reactive aura states after 4 sec (REACTIVE_TIMER_START).
const critReactiveWindow = time.Second * 4

func (hunter *Hunter) registerCritReactiveAuras() {
	needKillCommand := hunter.Talents.KillCommand && hunter.pet != nil
	needLacerate := hunter.Talents.Lacerate && hunter.Level >= 50
	if !needKillCommand && !needLacerate {
		return
	}

	if needKillCommand {
		hunter.KillCommandReadyAura = hunter.RegisterAura(core.Aura{
			Label:    "Kill Command Ready",
			Duration: critReactiveWindow,
		})
	}
	if needLacerate {
		hunter.LacerateReadyAura = hunter.RegisterAura(core.Aura{
			Label:    "Lacerate Ready",
			Duration: critReactiveWindow,
		})
	}

	core.MakePermanent(hunter.RegisterAura(core.Aura{
		Label: "Critical Strike Reactive Trigger",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !spell.ProcMask.Matches(core.ProcMaskMeleeOrRanged) || !result.DidCrit() {
				return
			}
			if hunter.KillCommandReadyAura != nil {
				hunter.KillCommandReadyAura.Activate(sim)
			}
			if hunter.LacerateReadyAura != nil {
				hunter.LacerateReadyAura.Activate(sim)
			}
		},
	}))
}
