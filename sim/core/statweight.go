package core

import (
	"math"
	"time"

	"github.com/isfir/wowsims-turtle/sim/core/proto"
	"github.com/isfir/wowsims-turtle/sim/core/simsignals"
	"github.com/isfir/wowsims-turtle/sim/core/stats"
	googleProto "google.golang.org/protobuf/proto"
)

const DTPSReferenceStat = stats.Armor

type UnitStats struct {
	Stats       stats.Stats
	PseudoStats []float64
}

func NewUnitStats() UnitStats {
	return UnitStats{
		PseudoStats: make([]float64, stats.PseudoStatsLen),
	}
}
func (s *UnitStats) AddStat(stat stats.UnitStat, value float64) {
	if stat.IsStat() {
		s.Stats[stat.StatIdx()] += value
	} else {
		s.PseudoStats[stat.PseudoStatIdx()] += value
	}
}
func (s *UnitStats) Get(stat stats.UnitStat) float64 {
	if stat.IsStat() {
		return s.Stats[stat.StatIdx()]
	} else {
		return s.PseudoStats[stat.PseudoStatIdx()]
	}
}

func (s *UnitStats) ToProto() *proto.UnitStats {
	return &proto.UnitStats{
		Stats:       s.Stats[:],
		PseudoStats: s.PseudoStats,
	}
}

type StatWeightValues struct {
	Weights       UnitStats
	WeightsStdev  UnitStats
	EpValues      UnitStats
	EpValuesStdev UnitStats

	// One-sided weights, from lowering (loss) or raising (gain) the stat only.
	WeightsLoss  UnitStats
	WeightsGain  UnitStats
	EpValuesLoss UnitStats
	EpValuesGain UnitStats
}

func NewStatWeightValues() StatWeightValues {
	return StatWeightValues{
		Weights:       NewUnitStats(),
		WeightsStdev:  NewUnitStats(),
		EpValues:      NewUnitStats(),
		EpValuesStdev: NewUnitStats(),
		WeightsLoss:   NewUnitStats(),
		WeightsGain:   NewUnitStats(),
		EpValuesLoss:  NewUnitStats(),
		EpValuesGain:  NewUnitStats(),
	}
}

func (swv *StatWeightValues) ToProto() *proto.StatWeightValues {
	return &proto.StatWeightValues{
		Weights:       swv.Weights.ToProto(),
		WeightsStdev:  swv.WeightsStdev.ToProto(),
		EpValues:      swv.EpValues.ToProto(),
		EpValuesStdev: swv.EpValuesStdev.ToProto(),
		EpValuesLoss:  swv.EpValuesLoss.ToProto(),
		EpValuesGain:  swv.EpValuesGain.ToProto(),
	}
}

type StatWeightsResult struct {
	Dps    StatWeightValues
	Hps    StatWeightValues
	Tps    StatWeightValues
	Dtps   StatWeightValues
	Tmi    StatWeightValues
	PDeath StatWeightValues
}

func NewStatWeightsResult() *StatWeightsResult {
	return &StatWeightsResult{
		Dps:    NewStatWeightValues(),
		Hps:    NewStatWeightValues(),
		Tps:    NewStatWeightValues(),
		Dtps:   NewStatWeightValues(),
		Tmi:    NewStatWeightValues(),
		PDeath: NewStatWeightValues(),
	}
}

func (swr *StatWeightsResult) ToProto() *proto.StatWeightsResult {
	return &proto.StatWeightsResult{
		Dps:    swr.Dps.ToProto(),
		Hps:    swr.Hps.ToProto(),
		Tps:    swr.Tps.ToProto(),
		Dtps:   swr.Dtps.ToProto(),
		Tmi:    swr.Tmi.ToProto(),
		PDeath: swr.PDeath.ToProto(),
	}
}

func buildStatWeightRequests(swr *proto.StatWeightsRequest) *proto.StatWeightRequestsData {
	if swr.Player.BonusStats == nil {
		swr.Player.BonusStats = &proto.UnitStats{}
	}
	if swr.Player.BonusStats.Stats == nil {
		swr.Player.BonusStats.Stats = make([]float64, stats.Len)
	}
	if swr.Player.BonusStats.PseudoStats == nil {
		swr.Player.BonusStats.PseudoStats = make([]float64, stats.PseudoStatsLen)
	}

	raidProto := SinglePlayerRaidProto(swr.Player, swr.PartyBuffs, swr.RaidBuffs, swr.Debuffs)
	raidProto.Tanks = swr.Tanks

	swr.SimOptions.SaveAllValues = true

	// Cut in half since we're doing above and below separately.
	// This number needs to be the same for the baseline sim too, so that RNG lines up perfectly.
	swr.SimOptions.Iterations /= 2

	// Make sure an RNG seed is always set because it gives more consistent results.
	// When there is no user-supplied seed it needs to be a randomly-selected seed
	// though, so that run-run differences still exist.
	if swr.SimOptions.RandomSeed == 0 {
		swr.SimOptions.RandomSeed = time.Now().UnixNano()
	}

	// Reduce variance even more by using test-level RNG controls.
	swr.SimOptions.UseLabeledRands = true

	swBaseResponse := &proto.StatWeightRequestsData{
		BaseRequest: &proto.RaidSimRequest{
			Raid:       raidProto,
			Encounter:  swr.Encounter,
			SimOptions: swr.SimOptions,
		},
		EpReferenceStat: swr.EpReferenceStat,
		StatSimRequests: []*proto.StatWeightsStatRequestData{},
	}

	// Do half the iterations with a positive, and half with a negative value for better accuracy.
	const defaultStatMod = 1.0 // lowered for SoD
	statModsLow := make([]float64, stats.UnitStatsLen)
	statModsHigh := make([]float64, stats.UnitStatsLen)

	// Make sure reference stat is included.
	statModsLow[swr.EpReferenceStat] = -defaultStatMod
	statModsHigh[swr.EpReferenceStat] = defaultStatMod

	statsToWeigh := stats.ProtoArrayToStatsList(swr.StatsToWeigh)
	for _, s := range statsToWeigh {
		stat := stats.UnitStatFromStat(s)
		statMod := defaultStatMod
		if stat.EqualsStat(stats.Armor) || stat.EqualsStat(stats.BonusArmor) || stat.EqualsStat(stats.Mana) {
			statMod = defaultStatMod * 20
		}
		statModsHigh[stat] = statMod
		statModsLow[stat] = -statMod
	}
	for _, s := range swr.PseudoStatsToWeigh {
		stat := stats.UnitStatFromPseudoStat(s)
		statMod := 3.0
		statModsHigh[stat] = statMod
		statModsLow[stat] = -statMod
	}

	for i := range statModsLow {
		stat := stats.UnitStatFromIdx(i)
		if statModsLow[stat] == 0 {
			continue
		}

		lowSimRequest := googleProto.Clone(swBaseResponse.BaseRequest).(*proto.RaidSimRequest)
		stat.AddToStatsProto(lowSimRequest.Raid.Parties[0].Players[0].BonusStats, statModsLow[stat])

		highSimRequest := googleProto.Clone(swBaseResponse.BaseRequest).(*proto.RaidSimRequest)
		stat.AddToStatsProto(highSimRequest.Raid.Parties[0].Players[0].BonusStats, statModsHigh[stat])

		swBaseResponse.StatSimRequests = append(swBaseResponse.StatSimRequests, &proto.StatWeightsStatRequestData{
			StatData: &proto.StatWeightsStatData{
				UnitStat: int32(stat),
				ModLow:   statModsLow[stat],
				ModHigh:  statModsHigh[stat],
			},
			RequestLow:  lowSimRequest,
			RequestHigh: highSimRequest,
		})
	}

	return swBaseResponse
}

func computeStatWeights(swcr *proto.StatWeightsCalcRequest) *proto.StatWeightsResult {
	haveRefStat := false
	for _, statResult := range swcr.StatSimResults {
		if statResult.StatData.UnitStat == int32(swcr.EpReferenceStat) {
			haveRefStat = true
			break
		}
	}
	if !haveRefStat {
		return &proto.StatWeightsResult{Error: &proto.ErrorOutcome{Message: "No result for reference stat exists!"}}
	}

	result := NewStatWeightsResult()

	// Per-iteration weight samples, kept so EP uncertainty can be computed from paired
	// differences against the reference stat (both share the same RNG streams).
	samples := map[*StatWeightValues]map[stats.UnitStat]*weightSamples{}
	for _, wv := range []*StatWeightValues{&result.Dps, &result.Hps, &result.Tps, &result.Dtps, &result.Tmi} {
		samples[wv] = map[stats.UnitStat]*weightSamples{}
	}

	for _, statResult := range swcr.StatSimResults {
		stat := stats.UnitStatFromIdx(int(statResult.StatData.UnitStat))

		baselinePlayer := swcr.BaseResult.RaidMetrics.Parties[0].Players[0]
		modPlayerLow := statResult.ResultLow.RaidMetrics.Parties[0].Players[0]
		modPlayerHigh := statResult.ResultHigh.RaidMetrics.Parties[0].Players[0]

		calcWeightResults := func(baselineMetrics *proto.DistributionMetrics, modLowMetrics *proto.DistributionMetrics, modHighMetrics *proto.DistributionMetrics, weightResults *StatWeightValues) {
			// RNG is fixed, so a side whose average is identical to the baseline did not change
			// anything, e.g. raising a stat that is already at its cap. Lowering it can still matter,
			// so measure the weight from whichever sides moved instead of discarding the stat.
			lowFlat := modLowMetrics.Avg == baselineMetrics.Avg
			highFlat := modHighMetrics.Avg == baselineMetrics.Avg
			if lowFlat && highFlat {
				return
			}

			n := len(baselineMetrics.AllValues)
			ws := &weightSamples{lo: make([]float64, n), hi: make([]float64, n), useLo: !lowFlat, useHi: !highFlat}
			var lo, hi aggregator
			for i := 0; i < n; i++ {
				ws.lo[i] = (modLowMetrics.AllValues[i] - baselineMetrics.AllValues[i]) / statResult.StatData.ModLow
				lo.add(ws.lo[i])
			}
			for i := 0; i < n; i++ {
				ws.hi[i] = (modHighMetrics.AllValues[i] - baselineMetrics.AllValues[i]) / statResult.StatData.ModHigh
				hi.add(ws.hi[i])
			}
			samples[weightResults][stat] = ws

			lossMean, _ := lo.meanAndStdDev()
			gainMean, _ := hi.meanAndStdDev()
			weightResults.WeightsLoss.AddStat(stat, lossMean)
			weightResults.WeightsGain.AddStat(stat, gainMean)

			var mean, stdev float64
			switch {
			// Weight stdevs are per sample and get divided by sqrt(total iterations) for display.
			// A one-sided weight only has half the samples, so scale its stdev to keep that correct.
			case highFlat:
				mean, stdev = lo.meanAndStdDev()
				stdev *= math.Sqrt2
			case lowFlat:
				mean, stdev = hi.meanAndStdDev()
				stdev *= math.Sqrt2
			default:
				mean, stdev = lo.merge(&hi).meanAndStdDev()
			}
			weightResults.Weights.AddStat(stat, mean)
			weightResults.WeightsStdev.AddStat(stat, stdev)
		}

		calcWeightResults(baselinePlayer.Dps, modPlayerLow.Dps, modPlayerHigh.Dps, &result.Dps)
		calcWeightResults(baselinePlayer.Hps, modPlayerLow.Hps, modPlayerHigh.Hps, &result.Hps)
		calcWeightResults(baselinePlayer.Threat, modPlayerLow.Threat, modPlayerHigh.Threat, &result.Tps)
		calcWeightResults(baselinePlayer.Dtps, modPlayerLow.Dtps, modPlayerHigh.Dtps, &result.Dtps)
		calcWeightResults(baselinePlayer.Tmi, modPlayerLow.Tmi, modPlayerHigh.Tmi, &result.Tmi)
		if modPlayerLow.ChanceOfDeath == baselinePlayer.ChanceOfDeath && modPlayerHigh.ChanceOfDeath == baselinePlayer.ChanceOfDeath {
			continue
		}
		meanLow := (modPlayerLow.ChanceOfDeath - baselinePlayer.ChanceOfDeath) / statResult.StatData.ModLow
		meanHigh := (modPlayerHigh.ChanceOfDeath - baselinePlayer.ChanceOfDeath) / statResult.StatData.ModHigh
		result.PDeath.Weights.AddStat(stat, (meanLow+meanHigh)/2)
		result.PDeath.WeightsStdev.AddStat(stat, 0)
	}

	referenceStat := stats.Stat(swcr.EpReferenceStat)

	// Compute EP results.
	for _, statData := range swcr.StatSimResults {
		stat := stats.UnitStatFromIdx(int(statData.StatData.UnitStat))

		calcEpResults := func(weightResults *StatWeightValues, refStat stats.Stat) {
			if weightResults.Weights.Stats[refStat] == 0 {
				return
			}
			refWeight := weightResults.Weights.Stats[refStat]
			mean := weightResults.Weights.Get(stat) / refWeight
			stdev := weightResults.WeightsStdev.Get(stat) / math.Abs(refWeight)
			if s, r := samples[weightResults][stat], samples[weightResults][stats.UnitStatFromStat(refStat)]; s != nil && r != nil {
				if sd, ok := s.epStdev(r, mean, refWeight); ok {
					stdev = sd
				}
			}
			weightResults.EpValues.AddStat(stat, mean)
			weightResults.EpValuesStdev.AddStat(stat, stdev)
			weightResults.EpValuesLoss.AddStat(stat, weightResults.WeightsLoss.Get(stat)/weightResults.Weights.Stats[refStat])
			weightResults.EpValuesGain.AddStat(stat, weightResults.WeightsGain.Get(stat)/weightResults.Weights.Stats[refStat])
		}

		calcEpResults(&result.Dps, referenceStat)
		calcEpResults(&result.Hps, referenceStat)
		calcEpResults(&result.Tps, referenceStat)
		calcEpResults(&result.Dtps, DTPSReferenceStat)
		calcEpResults(&result.Tmi, DTPSReferenceStat)
		calcEpResults(&result.PDeath, DTPSReferenceStat)
	}

	return result.ToProto()
}

// Run stat weight sims and compute weights.
func runStatWeights(request *proto.StatWeightsRequest, progress chan *proto.ProgressMetrics, signals simsignals.Signals) *proto.StatWeightsResult {
	requestData := buildStatWeightRequests(request)

	var iterationsTotal int32 = requestData.BaseRequest.SimOptions.Iterations
	var iterationsDone int32 = 0
	var simsTotal int32 = 1
	var simsCompleted int32 = 0

	for _, reqData := range requestData.StatSimRequests {
		iterationsTotal += reqData.RequestLow.SimOptions.Iterations
		iterationsTotal += reqData.RequestHigh.SimOptions.Iterations
		simsTotal += 2
	}

	waitForResult := func(srcProgressChannel chan *proto.ProgressMetrics) *proto.RaidSimResult {
		var lastCompleted int32 = 0
		for metrics := range srcProgressChannel {
			iterationsDone += metrics.CompletedIterations - lastCompleted
			lastCompleted = metrics.CompletedIterations

			if progress != nil {
				progress <- &proto.ProgressMetrics{
					TotalIterations:     iterationsTotal,
					CompletedIterations: iterationsDone,
					CompletedSims:       simsCompleted,
					TotalSims:           simsTotal,
				}
			}

			if metrics.FinalRaidResult != nil {
				simsCompleted++
				return metrics.FinalRaidResult
			}
		}
		return nil
	}

	simFunc := runSimConcurrent
	// Don't use go threads in wasm, it just adds more overhead and makes the worker more unresponsive.
	if IsRunningInWasm() || request.SimOptions.IsTest {
		simFunc = RunSim
	}

	baseProgress := make(chan *proto.ProgressMetrics, 100)
	go simFunc(requestData.BaseRequest, baseProgress, signals)
	baselineResult := waitForResult(baseProgress)
	if baselineResult.Error != nil {
		return &proto.StatWeightsResult{Error: baselineResult.Error}
	}

	statResults := []*proto.StatWeightsStatResultData{}

	for _, reqData := range requestData.StatSimRequests {
		lowProgress := make(chan *proto.ProgressMetrics, 100)
		go simFunc(reqData.RequestLow, lowProgress, signals)
		lowRes := waitForResult(lowProgress)
		if lowRes.Error != nil {
			return &proto.StatWeightsResult{Error: lowRes.Error}
		}

		highProgress := make(chan *proto.ProgressMetrics, 100)
		go simFunc(reqData.RequestHigh, highProgress, signals)
		highRes := waitForResult(highProgress)
		if highRes.Error != nil {
			return &proto.StatWeightsResult{Error: highRes.Error}
		}

		statResults = append(statResults, &proto.StatWeightsStatResultData{
			StatData:   reqData.StatData,
			ResultLow:  lowRes,
			ResultHigh: highRes,
		})
	}

	return computeStatWeights(&proto.StatWeightsCalcRequest{
		BaseResult:      baselineResult,
		EpReferenceStat: requestData.EpReferenceStat,
		StatSimResults:  statResults,
	})
}

type weightSamples struct {
	lo, hi       []float64 // per-iteration weight samples from the lowered / raised sims
	useLo, useHi bool      // which sides contribute to the weight
}

// Standard deviation (per sample, same convention as WeightsStdev) of EP = w/w_ref, using the
// delta method on paired samples: EP noise comes from (w_i - EP*ref_i), so noise that the stat
// and reference sims share through common RNG cancels out instead of being ignored.
func (s *weightSamples) epStdev(ref *weightSamples, ep float64, refWeight float64) (float64, bool) {
	if (s.useLo && !ref.useLo) || (s.useHi && !ref.useHi) || len(s.lo) != len(ref.lo) {
		return 0, false
	}
	var agg aggregator
	sides := 0
	if s.useLo {
		sides++
		for i := range s.lo {
			agg.add(s.lo[i] - ep*ref.lo[i])
		}
	}
	if s.useHi {
		sides++
		for i := range s.hi {
			agg.add(s.hi[i] - ep*ref.hi[i])
		}
	}
	_, sd := agg.meanAndStdDev()
	if sides == 1 {
		sd *= math.Sqrt2
	}
	return sd / math.Abs(refWeight), true
}
