package paladin

import (
	"time"

	"github.com/isfir/wowsims-turtle/sim/core"
	"github.com/isfir/wowsims-turtle/sim/core/stats"
)

// Libram IDs
const (
	SanctifiedOrb  = 20512
	LibramOfHope   = 22401
	LibramOfFervor = 23203
)

func init() {
	// Sanctified Orb is usable by any class; it is registered here, and every class package is linked into the real build.
	// Use: spell 24865 (client Spell.dbc and Turtle 1.17.1 spell_template agree): effect 1 SPELL_AURA_MOD_CRIT_PERCENT +5
	// (weapon crit, melee and ranged; the "$s1%" in the tooltip), effect 2 SPELL_AURA_MOD_SPELL_CRIT_CHANCE +3; 25 s.
	// Item cooldown 3 min, shared on-use item category 1141 for 25 s. The spell has StartRecoveryCategory 133 /
	// StartRecoveryTime 1500, the normal player GCD, and the server applies it to item casts, so using it costs a GCD.
	core.NewItemEffect(SanctifiedOrb, core.MakeTemporaryStatsOnUseCDRegistration(
		"ItemActive-20512",
		stats.Stats{stats.MeleeCrit: 5 * core.CritRatingPerCritChance, stats.SpellCrit: 3 * core.SpellCritRatingPerCritChance},
		time.Second*25,
		core.SpellConfig{
			ActionID: core.ActionID{ItemID: SanctifiedOrb},
			Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagOffensiveEquipment,
			Cast: core.CastConfig{
				DefaultCast: core.Cast{
					GCD: core.GCDDefault,
				},
			},
		},
		func(character *core.Character) core.Cooldown {
			return core.Cooldown{Timer: character.NewTimer(), Duration: time.Minute * 3}
		},
		func(character *core.Character) core.Cooldown {
			return core.Cooldown{Timer: character.GetOffensiveTrinketCD(), Duration: time.Second * 25}
		},
	))
}
