//go:build with_db

// EP accuracy harness for BM hunters: computes stat weights, ranks every usable item per slot by EP, and checks
// the top candidates against real swap sims (predicted vs. true DPS change, per-slot agreement, greedy EP sets).
// Skipped unless EP_COMPARE is set. See docs/octowow_mechanics.md.
// Run: EP_COMPARE=1 EP_LABEL=run EP_OUT=<dir> go test -tags=with_db ./sim/hunter -run 'TestEPCompare$' -v -timeout 0
// Knobs: EP_SW_ITERS, EP_TRUTH_ITERS, EP_TOPK, EP_ROUNDS, EP_SCORER (linear|capaware|twosided), EP_SCENARIO,
// EP_PSEUDO_SKILLS=0, EP_CODE (cache tag for stat weights). TestEPSeeds checks EP confidence intervals across seeds;
// TestEPProbe prints DPS for -2..+3 hit.
package hunter

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/isfir/wowsims-turtle/sim/core"
	"github.com/isfir/wowsims-turtle/sim/core/proto"
	"github.com/isfir/wowsims-turtle/sim/core/stats"
	googleProto "google.golang.org/protobuf/proto"
)

// Mirrors ui/hunter/sim.ts epStats / epPseudoStats / epReferenceStat.
var epcStats = []proto.Stat{
	proto.Stat_StatIntellect, proto.Stat_StatStrength, proto.Stat_StatAgility,
	proto.Stat_StatAttackPower, proto.Stat_StatRangedAttackPower, proto.Stat_StatMeleeHit,
	proto.Stat_StatMeleeCrit, proto.Stat_StatExpertise, proto.Stat_StatSpellPower,
	proto.Stat_StatSpellDamage, proto.Stat_StatNaturePower, proto.Stat_StatArcanePower,
	proto.Stat_StatSpellCrit, proto.Stat_StatMP5,
}
var epcPseudo = []proto.PseudoStat{
	proto.PseudoStat_PseudoStatMainHandDps, proto.PseudoStat_PseudoStatOffHandDps,
	proto.PseudoStat_PseudoStatRangedDps, proto.PseudoStat_PseudoStatMeleeSpeedMultiplier,
	proto.PseudoStat_PseudoStatRangedSpeedMultiplier,
	proto.PseudoStat_PseudoStatBowsSkill, proto.PseudoStat_PseudoStatCrossbowsSkill, proto.PseudoStat_PseudoStatGunsSkill,
}

var epcSkillPseudo = map[proto.WeaponSkill]proto.PseudoStat{
	proto.WeaponSkill_WeaponSkillAxes:            proto.PseudoStat_PseudoStatAxesSkill,
	proto.WeaponSkill_WeaponSkillSwords:          proto.PseudoStat_PseudoStatSwordsSkill,
	proto.WeaponSkill_WeaponSkillMaces:           proto.PseudoStat_PseudoStatMacesSkill,
	proto.WeaponSkill_WeaponSkillDaggers:         proto.PseudoStat_PseudoStatDaggersSkill,
	proto.WeaponSkill_WeaponSkillUnarmed:         proto.PseudoStat_PseudoStatUnarmedSkill,
	proto.WeaponSkill_WeaponSkillTwoHandedAxes:   proto.PseudoStat_PseudoStatTwoHandedAxesSkill,
	proto.WeaponSkill_WeaponSkillTwoHandedSwords: proto.PseudoStat_PseudoStatTwoHandedSwordsSkill,
	proto.WeaponSkill_WeaponSkillTwoHandedMaces:  proto.PseudoStat_PseudoStatTwoHandedMacesSkill,
	proto.WeaponSkill_WeaponSkillPolearms:        proto.PseudoStat_PseudoStatPolearmsSkill,
	proto.WeaponSkill_WeaponSkillStaves:          proto.PseudoStat_PseudoStatStavesSkill,
	proto.WeaponSkill_WeaponSkillThrown:          proto.PseudoStat_PseudoStatThrownSkill,
	proto.WeaponSkill_WeaponSkillBows:            proto.PseudoStat_PseudoStatBowsSkill,
	proto.WeaponSkill_WeaponSkillCrossbows:       proto.PseudoStat_PseudoStatCrossbowsSkill,
	proto.WeaponSkill_WeaponSkillGuns:            proto.PseudoStat_PseudoStatGunsSkill,
}

// EP_PSEUDO_SKILLS=0 drops weapon skills from weighing and item scoring (to reproduce pre-skill runs).
func epcUseSkills() bool { return os.Getenv("EP_PSEUDO_SKILLS") != "0" }

func epcPseudoList() []proto.PseudoStat {
	if epcUseSkills() {
		return epcPseudo
	}
	return epcPseudo[:5]
}

const epcRef = proto.Stat_StatRangedAttackPower

type epcScenario struct {
	Name     string
	Gear     string
	Rotation string
	Buffs    core.BuffsCombo
	Consumes *proto.Consumes
	BonusHit float64
}

func epcScenarios() []epcScenario {
	return []epcScenario{
		{Name: "bm_prebis_fullbuffs", Gear: "bm_prebis", Rotation: "bm", Buffs: core.FullBuffs, Consumes: P1Consumes.Consumes},
		{Name: "bm_prebis_minus2hit", Gear: "bm_prebis", Rotation: "bm", Buffs: core.FullBuffs, Consumes: P1Consumes.Consumes, BonusHit: -2},
		{Name: "bm_nobuffs", Gear: "bm_nobuffs", Rotation: "bm_nobuffs", Buffs: core.BuffsCombo{Label: "NoBuffs"}, Consumes: &proto.Consumes{}},
	}
}

func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		return n
	}
	return def
}

func epcPlayer(sc epcScenario, gear *proto.EquipmentSpec) *proto.Player {
	var bonus *proto.UnitStats
	if sc.BonusHit != 0 {
		bonus = &proto.UnitStats{Stats: make([]float64, stats.Len), PseudoStats: make([]float64, stats.PseudoStatsLen)}
		bonus.Stats[stats.MeleeHit] = sc.BonusHit
	}
	return core.WithSpec(&proto.Player{
		BonusStats:         bonus,
		Class:              proto.Class_ClassHunter,
		Race:               proto.Race_RaceOrc,
		Equipment:          gear,
		Consumes:           sc.Consumes,
		Buffs:              sc.Buffs.Player,
		TalentsString:      BMTalents,
		Profession1:        proto.Profession_Engineering,
		Rotation:           core.GetAplRotation("../../ui/hunter/apls", sc.Rotation).Rotation,
		DistanceFromTarget: 25,
		ReactionTimeMs:     150,
		ChannelClipDelayMs: 50,
	}, P1PlayerOptions)
}

func epcEncounter() *proto.Encounter { return core.MakeSingleTargetEncounter(0) }

var epcCache = map[string]json.RawMessage{}

func epcCachePath() string {
	if d := os.Getenv("EP_OUT"); d != "" {
		return filepath.Join(d, "cache.json")
	}
	return ""
}

func epcLoadCache() {
	if b, err := os.ReadFile(epcCachePath()); err == nil {
		json.Unmarshal(b, &epcCache)
	}
}

func epcSaveCache() {
	if p := epcCachePath(); p != "" {
		b, _ := json.Marshal(epcCache)
		os.WriteFile(p, b, 0o644)
	}
}

func epcGearKey(gear *proto.EquipmentSpec) string {
	k := ""
	for _, it := range gear.Items {
		k += fmt.Sprintf("%d:%d,", it.Id, it.Enchant)
	}
	return k
}

func epcDps(sc epcScenario, gear *proto.EquipmentSpec, iters int32) float64 {
	key := fmt.Sprintf("dps|%s|%d|%s", sc.Name, iters, epcGearKey(gear))
	if v, ok := epcCache[key]; ok {
		var f float64
		json.Unmarshal(v, &f)
		return f
	}
	f := epcDpsRaw(sc, gear, iters)
	b, _ := json.Marshal(f)
	epcCache[key] = b
	return f
}

func epcDpsRaw(sc epcScenario, gear *proto.EquipmentSpec, iters int32) float64 {
	raid := core.SinglePlayerRaidProto(epcPlayer(sc, gear), sc.Buffs.Party, sc.Buffs.Raid, sc.Buffs.Debuffs)
	res := core.RunRaidSimConcurrent(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  epcEncounter(),
		SimOptions: &proto.SimOptions{Iterations: iters, RandomSeed: 101, UseLabeledRands: true},
	})
	if res.Error != nil {
		panic(res.Error.Message)
	}
	return res.RaidMetrics.Dps.Avg
}

func epcFinalStats(sc epcScenario, gear *proto.EquipmentSpec) stats.Stats {
	raid := core.SinglePlayerRaidProto(epcPlayer(sc, gear), sc.Buffs.Party, sc.Buffs.Raid, sc.Buffs.Debuffs)
	r := core.ComputeStats(&proto.ComputeStatsRequest{Raid: raid, Encounter: epcEncounter()})
	return stats.FromFloatArray(r.RaidStats.Parties[0].Players[0].FinalStats.Stats)
}

type epcWeights struct {
	Ep      []float64 // stats then pseudo stats, indexed by UnitStat
	EpLoss  []float64
	EpGain  []float64
	EpCI    []float64
	RefDps  float64 // DPS per point of reference stat
	BaseDps float64
}

func epcComputeWeights(sc epcScenario, gear *proto.EquipmentSpec, iters int32) epcWeights {
	key := fmt.Sprintf("sw|%s|%s|%d|%s", os.Getenv("EP_CODE"), sc.Name, iters, epcGearKey(gear))
	if os.Getenv("EP_CODE") != "" {
		if v, ok := epcCache[key]; ok {
			var w epcWeights
			json.Unmarshal(v, &w)
			return w
		}
	}
	w := epcComputeWeightsRaw(sc, gear, iters)
	b, _ := json.Marshal(w)
	epcCache[key] = b
	epcSaveCache()
	return w
}

func epcComputeWeightsRaw(sc epcScenario, gear *proto.EquipmentSpec, iters int32) epcWeights {
	res := core.StatWeights(&proto.StatWeightsRequest{
		Player:             epcPlayer(sc, gear),
		RaidBuffs:          sc.Buffs.Raid,
		PartyBuffs:         sc.Buffs.Party,
		Debuffs:            sc.Buffs.Debuffs,
		Encounter:          epcEncounter(),
		SimOptions:         &proto.SimOptions{Iterations: iters, RandomSeed: 202},
		StatsToWeigh:       epcStats,
		PseudoStatsToWeigh: epcPseudoList(),
		EpReferenceStat:    epcRef,
	})
	if res.Error != nil {
		panic(res.Error.Message)
	}
	w := epcWeights{}
	w.Ep = append(append([]float64{}, res.Dps.EpValues.Stats...), res.Dps.EpValues.PseudoStats...)
	if res.Dps.EpValuesLoss != nil {
		w.EpLoss = append(append([]float64{}, res.Dps.EpValuesLoss.Stats...), res.Dps.EpValuesLoss.PseudoStats...)
		w.EpGain = append(append([]float64{}, res.Dps.EpValuesGain.Stats...), res.Dps.EpValuesGain.PseudoStats...)
	} else {
		w.EpLoss, w.EpGain = w.Ep, w.Ep
	}
	sd := append(append([]float64{}, res.Dps.EpValuesStdev.Stats...), res.Dps.EpValuesStdev.PseudoStats...)
	n := float64(iters)
	w.EpCI = make([]float64, len(sd))
	for i := range sd {
		w.EpCI[i] = 1.645 * sd[i] / math.Sqrt(n)
	}
	w.RefDps = res.Dps.Weights.Stats[epcRef]
	return w
}

// ---- item scoring (mirrors ui/core/player.ts computeItemEP) ----

type epcScorer func(item, cur core.Item, slot proto.ItemSlot, w epcWeights) float64

// Item as a UnitStat vector, mirroring ui/core/player.ts computeItemEP.
func epcVec(item core.Item, slot proto.ItemSlot) []float64 {
	v := make([]float64, int(stats.Len)+int(stats.PseudoStatsLen))
	if item.ID == 0 {
		return v
	}
	for i := 0; i < int(stats.Len); i++ {
		v[i] = item.Stats[i]
	}
	P := func(ps proto.PseudoStat) int { return int(stats.Len) + int(ps) }
	if item.SwingSpeed > 0 {
		dps := (item.WeaponDamageMin + item.WeaponDamageMax) / 2 / item.SwingSpeed
		switch slot {
		case proto.ItemSlot_ItemSlotMainHand:
			v[P(proto.PseudoStat_PseudoStatMainHandDps)] = dps
		case proto.ItemSlot_ItemSlotOffHand:
			v[P(proto.PseudoStat_PseudoStatOffHandDps)] = dps
		case proto.ItemSlot_ItemSlotRanged:
			v[P(proto.PseudoStat_PseudoStatRangedDps)] = dps
		}
	}
	v[P(proto.PseudoStat_BonusPhysicalDamage)] = item.BonusPhysicalDamage
	if h := item.Stats[stats.MeleeHaste]; h > 0 {
		v[P(proto.PseudoStat_PseudoStatMeleeSpeedMultiplier)] += h
		v[P(proto.PseudoStat_PseudoStatRangedSpeedMultiplier)] += h
	}
	v[stats.MeleeHaste] = 0
	if epcUseSkills() {
		for ws, ps := range epcSkillPseudo {
			v[P(ps)] += item.WeaponSkills[ws]
		}
	}
	return v
}

func dot(a, b []float64) float64 {
	t := 0.0
	for i := range a {
		t += a[i] * b[i]
	}
	return t
}

func epcLinear(item, _ core.Item, slot proto.ItemSlot, w epcWeights) float64 {
	return dot(epcVec(item, slot), w.Ep)
}

// Two-sided: stats gained relative to the equipped item use the gain weight, stats lost use the loss weight.
func epcTwoSided(item, cur core.Item, slot proto.ItemSlot, w epcWeights) float64 {
	iv, cv := epcVec(item, slot), epcVec(cur, slot)
	ep := dot(cv, w.Ep)
	for i := range iv {
		d := iv[i] - cv[i]
		if d > 0 {
			ep += d * w.EpGain[i]
		} else {
			ep += d * w.EpLoss[i]
		}
	}
	return ep
}

// Cap-aware: merged EP everywhere, except a stat whose gain side was flat (capped) gets 0 for gains.
func epcCapAware(item, cur core.Item, slot proto.ItemSlot, w epcWeights) float64 {
	iv, cv := epcVec(item, slot), epcVec(cur, slot)
	ep := dot(cv, w.Ep)
	for i := range iv {
		d := iv[i] - cv[i]
		if d > 0 && w.EpGain[i] == 0 {
			continue
		}
		ep += d * w.Ep[i]
	}
	return ep
}

var epcScorers = map[string]epcScorer{"linear": epcLinear, "twosided": epcTwoSided, "capaware": epcCapAware}

func epcSlotsFor(item core.Item) []proto.ItemSlot {
	switch item.Type {
	case proto.ItemType_ItemTypeHead:
		return []proto.ItemSlot{proto.ItemSlot_ItemSlotHead}
	case proto.ItemType_ItemTypeNeck:
		return []proto.ItemSlot{proto.ItemSlot_ItemSlotNeck}
	case proto.ItemType_ItemTypeShoulder:
		return []proto.ItemSlot{proto.ItemSlot_ItemSlotShoulder}
	case proto.ItemType_ItemTypeBack:
		return []proto.ItemSlot{proto.ItemSlot_ItemSlotBack}
	case proto.ItemType_ItemTypeChest:
		return []proto.ItemSlot{proto.ItemSlot_ItemSlotChest}
	case proto.ItemType_ItemTypeWrist:
		return []proto.ItemSlot{proto.ItemSlot_ItemSlotWrist}
	case proto.ItemType_ItemTypeHands:
		return []proto.ItemSlot{proto.ItemSlot_ItemSlotHands}
	case proto.ItemType_ItemTypeWaist:
		return []proto.ItemSlot{proto.ItemSlot_ItemSlotWaist}
	case proto.ItemType_ItemTypeLegs:
		return []proto.ItemSlot{proto.ItemSlot_ItemSlotLegs}
	case proto.ItemType_ItemTypeFeet:
		return []proto.ItemSlot{proto.ItemSlot_ItemSlotFeet}
	case proto.ItemType_ItemTypeFinger:
		return []proto.ItemSlot{proto.ItemSlot_ItemSlotFinger1, proto.ItemSlot_ItemSlotFinger2}
	case proto.ItemType_ItemTypeTrinket:
		return []proto.ItemSlot{proto.ItemSlot_ItemSlotTrinket1, proto.ItemSlot_ItemSlotTrinket2}
	case proto.ItemType_ItemTypeRanged:
		return []proto.ItemSlot{proto.ItemSlot_ItemSlotRanged}
	case proto.ItemType_ItemTypeWeapon:
		switch item.HandType {
		case proto.HandType_HandTypeTwoHand, proto.HandType_HandTypeMainHand:
			return []proto.ItemSlot{proto.ItemSlot_ItemSlotMainHand}
		case proto.HandType_HandTypeOffHand:
			return []proto.ItemSlot{proto.ItemSlot_ItemSlotOffHand}
		case proto.HandType_HandTypeOneHand:
			return []proto.ItemSlot{proto.ItemSlot_ItemSlotMainHand, proto.ItemSlot_ItemSlotOffHand}
		}
	}
	return nil
}

func epcCandidates(slot proto.ItemSlot, gear *proto.EquipmentSpec) []core.Item {
	filter := ItemFilters
	filter.Class = proto.Class_ClassHunter
	mh := core.ItemsByID[gear.Items[proto.ItemSlot_ItemSlotMainHand].Id]
	if slot == proto.ItemSlot_ItemSlotOffHand && mh.HandType == proto.HandType_HandTypeTwoHand {
		return nil
	}
	var other int32
	switch slot {
	case proto.ItemSlot_ItemSlotFinger1:
		other = gear.Items[proto.ItemSlot_ItemSlotFinger2].Id
	case proto.ItemSlot_ItemSlotFinger2:
		other = gear.Items[proto.ItemSlot_ItemSlotFinger1].Id
	case proto.ItemSlot_ItemSlotTrinket1:
		other = gear.Items[proto.ItemSlot_ItemSlotTrinket2].Id
	case proto.ItemSlot_ItemSlotTrinket2:
		other = gear.Items[proto.ItemSlot_ItemSlotTrinket1].Id
	case proto.ItemSlot_ItemSlotMainHand:
		other = gear.Items[proto.ItemSlot_ItemSlotOffHand].Id
	case proto.ItemSlot_ItemSlotOffHand:
		other = gear.Items[proto.ItemSlot_ItemSlotMainHand].Id
	}
	var out []core.Item
	for _, it := range core.ItemsByID {
		if !filter.Matches(it, true) || it.ID == other {
			continue
		}
		for _, s := range epcSlotsFor(it) {
			if s == slot {
				out = append(out, it)
			}
		}
	}
	return out
}

// True when the item is already worn in the paired ring/trinket/hand slot.
func epcDupe(gear *proto.EquipmentSpec, slot proto.ItemSlot, id int32) bool {
	pair := map[proto.ItemSlot]proto.ItemSlot{
		proto.ItemSlot_ItemSlotFinger2: proto.ItemSlot_ItemSlotFinger1, proto.ItemSlot_ItemSlotFinger1: proto.ItemSlot_ItemSlotFinger2,
		proto.ItemSlot_ItemSlotTrinket2: proto.ItemSlot_ItemSlotTrinket1, proto.ItemSlot_ItemSlotTrinket1: proto.ItemSlot_ItemSlotTrinket2,
		proto.ItemSlot_ItemSlotOffHand: proto.ItemSlot_ItemSlotMainHand, proto.ItemSlot_ItemSlotMainHand: proto.ItemSlot_ItemSlotOffHand,
	}
	o, ok := pair[slot]
	return ok && gear.Items[o].Id == id
}

func epcSwap(gear *proto.EquipmentSpec, slot proto.ItemSlot, id int32) *proto.EquipmentSpec {
	g := googleProto.Clone(gear).(*proto.EquipmentSpec)
	g.Items[slot] = &proto.ItemSpec{Id: id, Enchant: gear.Items[slot].Enchant}
	if slot == proto.ItemSlot_ItemSlotMainHand && core.ItemsByID[id].HandType == proto.HandType_HandTypeTwoHand {
		g.Items[proto.ItemSlot_ItemSlotOffHand] = &proto.ItemSpec{}
	}
	return g
}

// ---- output ----

type epcCand struct {
	ID        int32
	Name      string
	EP        float64
	PredDelta float64 // (EP - EP(current)) * RefDps
	TrueDelta float64
	HasEffect bool
}

type epcSlotResult struct {
	Slot     string
	Current  string
	Cands    []epcCand
	EPPick   int32 // best EP (incl. current)
	TruePick int32 // best true delta among simmed candidates (incl. current)
}

type epcRound struct {
	Gear    []int32
	Dps     float64
	Changed []string
}

type epcScenarioResult struct {
	Scenario   string
	Scorer     string
	BaseDps    float64
	Hit        float64
	Crit       float64
	EP         map[string][4]float64
	RefDps     float64
	Slots      []epcSlotResult
	SlotAgree  int
	SlotTotal  int
	MAE        float64
	Spearman   float64
	EPSetDps   float64 // greedy EP picks applied simultaneously
	TrueSetDps float64 // truth picks among top-K applied simultaneously
	Rounds     []epcRound
}

func epcStatName(i int) string {
	if i < int(stats.Len) {
		return stats.Stat(i).StatName()
	}
	return proto.PseudoStat(i - int(stats.Len)).String()
}

func spearman(a, b []float64) float64 {
	rank := func(x []float64) []float64 {
		idx := make([]int, len(x))
		for i := range idx {
			idx[i] = i
		}
		sort.Slice(idx, func(i, j int) bool { return x[idx[i]] < x[idx[j]] })
		r := make([]float64, len(x))
		for k, i := range idx {
			r[i] = float64(k)
		}
		return r
	}
	ra, rb := rank(a), rank(b)
	n := float64(len(a))
	var d2 float64
	for i := range ra {
		d2 += (ra[i] - rb[i]) * (ra[i] - rb[i])
	}
	if n < 2 {
		return 0
	}
	return 1 - 6*d2/(n*(n*n-1))
}

func TestEPCompare(t *testing.T) {
	if os.Getenv("EP_COMPARE") == "" {
		t.Skip("set EP_COMPARE=1")
	}
	label := os.Getenv("EP_LABEL")
	if label == "" {
		label = "run"
	}
	outDir := os.Getenv("EP_OUT")
	swIters := int32(envInt("EP_SW_ITERS", 20000))
	truthIters := int32(envInt("EP_TRUTH_ITERS", 20000))
	topK := envInt("EP_TOPK", 6)
	rounds := envInt("EP_ROUNDS", 3)
	scorerName := os.Getenv("EP_SCORER")
	if scorerName == "" {
		scorerName = "linear"
	}
	scorer := epcScorers[scorerName]

	epcLoadCache()
	defer epcSaveCache()
	var all []epcScenarioResult
	for _, sc := range epcScenarios() {
		if f := os.Getenv("EP_SCENARIO"); f != "" && f != sc.Name {
			continue
		}
		base := core.GetGearSet("../../ui/hunter/gear_sets", sc.Gear).GearSet
		r := epcScenarioResult{Scenario: sc.Name, Scorer: scorerName, EP: map[string][4]float64{}}
		r.BaseDps = epcDps(sc, base, truthIters)
		fs := epcFinalStats(sc, base)
		r.Hit, r.Crit = fs[stats.MeleeHit], fs[stats.MeleeCrit]
		w := epcComputeWeights(sc, base, swIters)
		r.RefDps = w.RefDps
		for i := range w.Ep {
			if w.Ep[i] != 0 || w.EpCI[i] != 0 {
				r.EP[epcStatName(i)] = [4]float64{w.Ep[i], w.EpCI[i], w.EpLoss[i], w.EpGain[i]}
			}
		}
		t.Logf("[%s] base dps %.1f hit %.2f crit %.2f refDps %.4f", sc.Name, r.BaseDps, r.Hit, r.Crit, r.RefDps)

		var preds, truths []float64
		var absErr float64
		epGear := googleProto.Clone(base).(*proto.EquipmentSpec)
		trueGear := googleProto.Clone(base).(*proto.EquipmentSpec)
		for slot := proto.ItemSlot_ItemSlotHead; slot <= proto.ItemSlot_ItemSlotRanged; slot++ {
			curID := base.Items[slot].Id
			cands := epcCandidates(slot, base)
			if len(cands) == 0 {
				continue
			}
			curEP := 0.0
			if curID != 0 {
				curEP = scorer(core.ItemsByID[curID], core.ItemsByID[curID], slot, w)
			}
			type sc2 struct {
				it core.Item
				ep float64
			}
			scored := make([]sc2, 0, len(cands))
			for _, it := range cands {
				scored = append(scored, sc2{it, scorer(it, core.ItemsByID[curID], slot, w)})
			}
			sort.Slice(scored, func(i, j int) bool {
				if scored[i].ep != scored[j].ep {
					return scored[i].ep > scored[j].ep
				}
				return scored[i].it.ID < scored[j].it.ID
			})
			sr := epcSlotResult{Slot: slot.String(), Current: core.ItemsByID[curID].Name}
			sr.EPPick, sr.TruePick = curID, curID
			bestEP, bestTrue := curEP, 0.0
			for k := 0; k < len(scored) && k < topK; k++ {
				c := scored[k]
				cd := epcCand{ID: c.it.ID, Name: c.it.Name, EP: c.ep, PredDelta: (c.ep - curEP) * w.RefDps, HasEffect: core.HasItemEffect(c.it.ID)}
				if c.it.ID == curID {
					cd.TrueDelta = 0
				} else {
					cd.TrueDelta = epcDps(sc, epcSwap(base, slot, c.it.ID), truthIters) - r.BaseDps
				}
				if c.ep > bestEP {
					bestEP, sr.EPPick = c.ep, c.it.ID
				}
				if cd.TrueDelta > bestTrue {
					bestTrue, sr.TruePick = cd.TrueDelta, c.it.ID
				}
				if c.it.ID != curID {
					preds = append(preds, cd.PredDelta)
					truths = append(truths, cd.TrueDelta)
					absErr += math.Abs(cd.PredDelta - cd.TrueDelta)
				}
				sr.Cands = append(sr.Cands, cd)
			}
			r.SlotTotal++
			if sr.EPPick == sr.TruePick {
				r.SlotAgree++
			}
			if sr.EPPick != curID && !epcDupe(epGear, slot, sr.EPPick) {
				epGear = epcSwap(epGear, slot, sr.EPPick)
			}
			if sr.TruePick != curID && !epcDupe(trueGear, slot, sr.TruePick) {
				trueGear = epcSwap(trueGear, slot, sr.TruePick)
			}
			r.Slots = append(r.Slots, sr)
		}
		if len(preds) > 0 {
			r.MAE = absErr / float64(len(preds))
			r.Spearman = spearman(preds, truths)
		}
		r.EPSetDps = epcDps(sc, epGear, truthIters)
		r.TrueSetDps = epcDps(sc, trueGear, truthIters)
		t.Logf("[%s] slot agree %d/%d MAE %.2f spearman %.3f epSet %.1f trueSet %.1f", sc.Name, r.SlotAgree, r.SlotTotal, r.MAE, r.Spearman, r.EPSetDps, r.TrueSetDps)

		// Iterated greedy: recompute EP at the new gear, re-pick best-EP per slot from the full pool.
		g := googleProto.Clone(base).(*proto.EquipmentSpec)
		for rd := 0; rd < rounds; rd++ {
			wr := epcComputeWeights(sc, g, swIters)
			var changed []string
			for slot := proto.ItemSlot_ItemSlotHead; slot <= proto.ItemSlot_ItemSlotRanged; slot++ {
				cands := epcCandidates(slot, g)
				if len(cands) == 0 {
					continue
				}
				curID := g.Items[slot].Id
				best, bestEP := curID, -1e9
				if curID != 0 {
					bestEP = scorer(core.ItemsByID[curID], core.ItemsByID[curID], slot, wr)
				}
				for _, it := range cands {
					if e := scorer(it, core.ItemsByID[curID], slot, wr); e > bestEP+1e-9 {
						best, bestEP = it.ID, e
					}
				}
				if best != curID {
					changed = append(changed, fmt.Sprintf("%s: %s -> %s", slot, core.ItemsByID[curID].Name, core.ItemsByID[best].Name))
					g = epcSwap(g, slot, best)
				}
			}
			ids := make([]int32, len(g.Items))
			for i, it := range g.Items {
				ids[i] = it.Id
			}
			round := epcRound{Gear: ids, Dps: epcDps(sc, g, truthIters), Changed: changed}
			r.Rounds = append(r.Rounds, round)
			t.Logf("[%s] round %d dps %.1f changes %d", sc.Name, rd+1, round.Dps, len(changed))
			if len(changed) == 0 {
				break
			}
		}
		all = append(all, r)
		epcSaveCache()
	}

	if outDir != "" {
		b, _ := json.MarshalIndent(all, "", " ")
		if err := os.WriteFile(filepath.Join(outDir, label+".json"), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEPProbe(t *testing.T) {
	if os.Getenv("EP_COMPARE") == "" {
		t.Skip()
	}
	sc := epcScenarios()[0]
	base := core.GetGearSet("../../ui/hunter/gear_sets", sc.Gear).GearSet
	for _, d := range []float64{-2, -1, 0, 1, 2, 3} {
		p := epcPlayer(sc, base)
		p.BonusStats = &proto.UnitStats{Stats: make([]float64, stats.Len), PseudoStats: make([]float64, stats.PseudoStatsLen)}
		p.BonusStats.Stats[stats.MeleeHit] = d
		raid := core.SinglePlayerRaidProto(p, sc.Buffs.Party, sc.Buffs.Raid, sc.Buffs.Debuffs)
		res := core.RunRaidSim(&proto.RaidSimRequest{Raid: raid, Encounter: epcEncounter(),
			SimOptions: &proto.SimOptions{Iterations: 2000, RandomSeed: 5, UseLabeledRands: true}})
		fs := epcFinalStats(sc, base)
		raid0 := core.SinglePlayerRaidProto(epcPlayer(sc, base), sc.Buffs.Party, sc.Buffs.Raid, sc.Buffs.Debuffs)
		r0 := core.ComputeStats(&proto.ComputeStatsRequest{Raid: raid0, Encounter: epcEncounter()})
		ps := r0.RaidStats.Parties[0].Players[0].FinalStats.PseudoStats
		t.Logf("pseudo rangedHit %.2f all %v", ps[proto.PseudoStat_PseudoStatRangedHit], ps)
		t.Logf("hit %+v -> dps %.3f (gear hit %.1f)", d, res.RaidMetrics.Dps.Avg, fs[stats.MeleeHit])
	}
}

func TestEPSeeds(t *testing.T) {
	if os.Getenv("EP_COMPARE") == "" {
		t.Skip()
	}
	sc := epcScenarios()[0]
	base := core.GetGearSet("../../ui/hunter/gear_sets", sc.Gear).GearSet
	iters := int32(envInt("EP_SW_ITERS", 4000))
	type acc struct{ vals, ses []float64 }
	res := map[string]*acc{}
	for seed := int64(1); seed <= int64(envInt("EP_SEEDS", 6)); seed++ {
		r := core.StatWeights(&proto.StatWeightsRequest{
			Player: epcPlayer(sc, base), RaidBuffs: sc.Buffs.Raid, PartyBuffs: sc.Buffs.Party, Debuffs: sc.Buffs.Debuffs,
			Encounter: epcEncounter(), SimOptions: &proto.SimOptions{Iterations: iters, RandomSeed: seed * 7919},
			StatsToWeigh: epcStats, PseudoStatsToWeigh: epcPseudoList(), EpReferenceStat: epcRef,
		})
		ep := append(append([]float64{}, r.Dps.EpValues.Stats...), r.Dps.EpValues.PseudoStats...)
		sd := append(append([]float64{}, r.Dps.EpValuesStdev.Stats...), r.Dps.EpValuesStdev.PseudoStats...)
		for i := range ep {
			if ep[i] == 0 {
				continue
			}
			n := epcStatName(i)
			if res[n] == nil {
				res[n] = &acc{}
			}
			res[n].vals = append(res[n].vals, ep[i])
			res[n].ses = append(res[n].ses, sd[i]/math.Sqrt(float64(iters)))
		}
	}
	for n, a := range res {
		var m, v, se float64
		for _, x := range a.vals {
			m += x
		}
		m /= float64(len(a.vals))
		for i, x := range a.vals {
			v += (x - m) * (x - m)
			se += a.ses[i]
		}
		v /= float64(len(a.vals) - 1)
		se /= float64(len(a.vals))
		t.Logf("%-34s mean %7.3f  seed-spread sd %6.3f  reported se %6.3f  ratio %.2f", n, m, math.Sqrt(v), se, math.Sqrt(v)/se)
	}
}
