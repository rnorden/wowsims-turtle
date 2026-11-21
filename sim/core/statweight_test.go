package core

import (
	"math"
	"testing"

	"github.com/isfir/wowsims-turtle/sim/core/proto"
	"github.com/isfir/wowsims-turtle/sim/core/stats"
)

func swTestResult(dps []float64) *proto.RaidSimResult {
	sum := 0.0
	for _, v := range dps {
		sum += v
	}
	return &proto.RaidSimResult{RaidMetrics: &proto.RaidMetrics{Parties: []*proto.PartyMetrics{{Players: []*proto.UnitMetrics{{
		Dps:    &proto.DistributionMetrics{Avg: sum / float64(len(dps)), AllValues: dps},
		Hps:    &proto.DistributionMetrics{AllValues: make([]float64, len(dps))},
		Threat: &proto.DistributionMetrics{AllValues: make([]float64, len(dps))},
		Dtps:   &proto.DistributionMetrics{AllValues: make([]float64, len(dps))},
		Tmi:    &proto.DistributionMetrics{AllValues: make([]float64, len(dps))},
	}}}}}}
}

func swTestStat(stat stats.Stat, mod float64, base []float64, lowDelta, highDelta []float64) *proto.StatWeightsStatResultData {
	low := make([]float64, len(base))
	high := make([]float64, len(base))
	for i := range base {
		low[i] = base[i] + lowDelta[i]
		high[i] = base[i] + highDelta[i]
	}
	return &proto.StatWeightsStatResultData{
		StatData:   &proto.StatWeightsStatData{UnitStat: int32(stat), ModLow: -mod, ModHigh: mod},
		ResultLow:  swTestResult(low),
		ResultHigh: swTestResult(high),
	}
}

func TestComputeStatWeightsCappedStat(t *testing.T) {
	base := []float64{1000, 1010, 990, 1005}
	zero := []float64{0, 0, 0, 0}

	res := computeStatWeights(&proto.StatWeightsCalcRequest{
		BaseResult:      swTestResult(base),
		EpReferenceStat: proto.Stat_StatAttackPower,
		StatSimResults: []*proto.StatWeightsStatResultData{
			// Reference: 0.5 DPS per point, with per-iteration noise.
			swTestStat(stats.AttackPower, 1, base, []float64{-0.4, -0.6, -0.5, -0.5}, []float64{0.6, 0.4, 0.5, 0.5}),
			// Hit at its cap: raising it changes nothing, lowering it costs 10 DPS.
			swTestStat(stats.MeleeHit, 1, base, []float64{-10, -10, -10, -10}, zero),
			// A stat that does nothing at all.
			swTestStat(stats.Intellect, 1, base, zero, zero),
		},
	})

	ep := res.Dps.EpValues.Stats
	loss := res.Dps.EpValuesLoss.Stats
	gain := res.Dps.EpValuesGain.Stats
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

	if !near(ep[stats.AttackPower], 1) {
		t.Errorf("reference EP = %v, want 1", ep[stats.AttackPower])
	}
	if !near(ep[stats.MeleeHit], 20) {
		t.Errorf("capped hit EP = %v, want 20 (from the lowered side only)", ep[stats.MeleeHit])
	}
	if !near(loss[stats.MeleeHit], 20) || !near(gain[stats.MeleeHit], 0) {
		t.Errorf("capped hit loss/gain = %v/%v, want 20/0", loss[stats.MeleeHit], gain[stats.MeleeHit])
	}
	if ep[stats.Intellect] != 0 || res.Dps.Weights.Stats[stats.Intellect] != 0 {
		t.Errorf("no-op stat should have 0 weight, got %v", ep[stats.Intellect])
	}
	if res.Dps.EpValuesStdev.Stats[stats.AttackPower] != 0 {
		t.Errorf("reference stat EP stdev = %v, want 0 (EP is exactly 1)", res.Dps.EpValuesStdev.Stats[stats.AttackPower])
	}
}
