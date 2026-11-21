package hunter

import (
	"math"
	"testing"

	"github.com/isfir/wowsims-turtle/sim/core"
	"github.com/isfir/wowsims-turtle/sim/core/proto"
	"github.com/isfir/wowsims-turtle/sim/core/simsignals"
	"github.com/isfir/wowsims-turtle/sim/core/stats"
)

// Arcane Shot's flat Arcane damage scales with Arcane + generic spell power at 0.429 (rank 8).
func TestArcaneShotSpellPowerCoefficient(t *testing.T) {
	const base = 100.0

	hit := func(arcanePower, spellPower float64) float64 {
		bonus := &proto.UnitStats{Stats: make([]float64, stats.Len), PseudoStats: make([]float64, stats.PseudoStatsLen)}
		bonus.Stats[stats.ArcanePower] = arcanePower
		bonus.Stats[stats.SpellPower] = spellPower
		player := core.WithSpec(&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Equipment:          &proto.EquipmentSpec{},
			BonusStats:         bonus,
			TalentsString:      BMTalents,
			DistanceFromTarget: 25,
		}, P1PlayerOptions)
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid:       core.SinglePlayerRaidProto(player, nil, nil, nil),
			Encounter:  core.MakeSingleTargetEncounter(0),
			SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1},
		}, simsignals.CreateSignals())
		sim.Reset()

		hunter := sim.Raid.Parties[0].Players[0].(HunterAgent).GetHunter()
		spell := hunter.ArcaneShot
		if spell.Rank != 8 {
			t.Fatalf("expected Arcane Shot rank 8, got %d", spell.Rank)
		}
		// Same seed in both sims, so any resist roll is identical and only the SP term differs.
		return spell.CalcDamage(sim, sim.Encounter.TargetUnits[0], base, spell.OutcomeAlwaysHit).Damage
	}

	noSP := hit(0, 0)
	withSP := hit(60, 40)
	got := (withSP/noSP - 1) * base / 100
	if math.Abs(got-0.429) > 1e-9 {
		t.Fatalf("Arcane Shot SP coefficient = %.6f, want 0.429 (hit %.3f -> %.3f)", got, noSP, withSP)
	}
}
