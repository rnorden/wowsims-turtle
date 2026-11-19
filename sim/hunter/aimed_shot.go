package hunter

import (
	"time"

	"github.com/isfir/wowsims-turtle/sim/core"
	"github.com/isfir/wowsims-turtle/sim/core/proto"
)

func (hunter *Hunter) getAimedShotConfig(rank int) core.SpellConfig {
	spellId := [7]int32{0, 19434, 20900, 20901, 20902, 20903, 20904}[rank]
	baseDamage := [7]float64{0, 70, 125, 200, 330, 460, 600}[rank]
	manaCost := [7]float64{0, 50, 75, 105, 140, 170, 205}[rank]
	level := [7]int{0, 20, 28, 36, 44, 52, 60}[rank]

	return core.SpellConfig{
		SpellCode:     SpellCode_HunterAimedShot,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeRanged,
		ProcMask:      core.ProcMaskRangedSpecial,
		Flags:         core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagShot,
		CastType:      proto.CastType_CastTypeRanged,
		Rank:          rank,
		RequiredLevel: level,
		MissileSpeed:  40,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      time.Second,
				CastTime: time.Second * 2,
			},
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: time.Second*26 - time.Second*2*time.Duration(hunter.Talents.Swiftshot),
			},
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				cast.CastTime = spell.CastTime()
				hunter.Unit.AutoAttacks.CancelAutoSwing(sim)
			},
			IgnoreHaste: true,
			CastTime:    hunter.shotCastTime,
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget >= core.MinRangedAttackDistance
		},

		CritDamageBonus: hunter.mortalShots(),

		DamageMultiplier: 1 + 0.05*float64(hunter.Talents.ImprovedMarksmanship),
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := hunter.rangedWeaponDamage(sim, spell, target, true) + baseDamage

			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeRangedHitAndCrit)
			hunter.Unit.AutoAttacks.EnableAutoSwing(sim)
			spell.WaitTravelTime(sim, func(s *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	}
}

func (hunter *Hunter) registerAimedShotSpell() {
	if !hunter.Talents.AimedShot {
		return
	}

	for rank := 1; rank <= 6; rank++ {
		config := hunter.getAimedShotConfig(rank)

		if config.RequiredLevel <= int(hunter.Level) {
			hunter.AimedShot = hunter.GetOrRegisterSpell(config)
		}
	}
}
