package hunter

import (
	"time"

	"github.com/isfir/wowsims-turtle/sim/core"
)

// Lacerate (Survival talent): usable after a critical strike. Deals 40% of melee attack power and
// makes the target bleed for 20% of the damage dealt over 8 sec.
func (hunter *Hunter) registerLacerateSpell() {
	if hunter.LacerateReadyAura == nil {
		return
	}

	hunter.LacerateBleed = hunter.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_HunterLacerateBleed,
		ActionID:    core.ActionID{SpellID: 48050},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell | core.SpellFlagIgnoreAttackerModifiers,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Lacerate Bleed",
			},
			NumberOfTicks: 4,
			TickLength:    time.Second * 2,

			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},
	})

	hunter.Lacerate = hunter.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_HunterLacerate,
		ActionID:    core.ActionID{SpellID: 48049},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagStrike,

		ManaCost: core.ManaCostOptions{
			BaseCost: 0.08,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: time.Second * 10,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget <= core.MaxMeleeAttackDistance && hunter.LacerateReadyAura.IsActive()
		},

		BonusCritRating:  hunter.savageStrikesCritRating(),
		CritDamageBonus:  hunter.killerInstinctCritDamageBonus(),
		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			hunter.LacerateReadyAura.Deactivate(sim)

			damage := 0.4 * spell.MeleeAttackPower(target)
			result := spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMeleeSpecialHitAndCrit)

			if result.Landed() {
				dot := hunter.LacerateBleed.Dot(target)
				dot.SnapshotBaseDamage = result.Damage * 0.2 / float64(dot.OriginalNumberOfTicks)
				dot.SnapshotAttackerMultiplier = 1
				dot.Apply(sim)
			}
		},
	})
}
