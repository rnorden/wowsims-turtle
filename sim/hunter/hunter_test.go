package hunter

import (
	"testing"

	_ "github.com/isfir/wowsims-turtle/sim/common" // imported to get item effects included.
	"github.com/isfir/wowsims-turtle/sim/core"
	"github.com/isfir/wowsims-turtle/sim/core/proto"
)

func init() {
	RegisterHunter()
}

func TestBMHunter(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:      proto.Class_ClassHunter,
			Phase:      1,
			Race:       proto.Race_RaceOrc,
			OtherRaces: []proto.Race{proto.Race_RaceDwarf},

			Talents:  BMTalents,
			GearSet:  core.GetGearSet("../../ui/hunter/gear_sets", "bm_prebis"),
			Rotation: core.GetAplRotation("../../ui/hunter/apls", "bm"),
			OtherRotations: []core.RotationCombo{
				core.GetAplRotation("../../ui/hunter/apls", "bm_nobuffs"),
			},
			Buffs:       core.FullBuffs,
			Consumes:    P1Consumes,
			SpecOptions: core.SpecOptionsCombo{Label: "Hunter", SpecOptions: P1PlayerOptions},

			// Shots require at least core.MinRangedAttackDistance.
			DistanceFromTarget: 25,

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatAttackPower,
			StatsToWeigh:    Stats,
		},
	}))
}

func TestMMHunter(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class: proto.Class_ClassHunter,
			Phase: 1,
			Race:  proto.Race_RaceOrc,

			Talents:     MMTalents,
			GearSet:     core.GetGearSet("../../ui/hunter/gear_sets", "p0.bis"),
			Rotation:    core.GetAplRotation("../../ui/hunter/apls", "p1"),
			Buffs:       core.FullBuffs,
			Consumes:    P1Consumes,
			SpecOptions: core.SpecOptionsCombo{Label: "Hunter", SpecOptions: P1PlayerOptions},

			// Shots require at least core.MinRangedAttackDistance.
			DistanceFromTarget: 25,

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatAttackPower,
			StatsToWeigh:    Stats,
		},
	}))
}

// Must match ui/hunter/presets.ts.
var BMTalents = "5500000152053102251-0505013"
var MMTalents = "550000005-0505113250123251"

var P1Consumes = core.ConsumesCombo{
	Label: "P1-Consumes",
	Consumes: &proto.Consumes{
		AgilityElixir:   proto.AgilityElixir_ElixirOfTheMongoose,
		AttackPowerBuff: proto.AttackPowerBuff_JujuMight,
		DefaultPotion:   proto.Potions_ManaPotion,
		Flask:           proto.Flask_FlaskOfSupremePower,
		Food:            proto.Food_FoodSagefishDelight,
		MainHandImbue:   proto.WeaponImbue_Windfury,
		OffHandImbue:    proto.WeaponImbue_ElementalSharpeningStone,
		SpellPowerBuff:  proto.SpellPowerBuff_GreaterArcaneElixir,
		StrengthBuff:    proto.StrengthBuff_JujuPower,
	},
}

var P1PlayerOptions = &proto.Player_Hunter{
	Hunter: &proto.Hunter{
		Options: &proto.Hunter_Options{
			Ammo:           proto.Hunter_Options_RazorArrow,
			PetType:        proto.Hunter_Options_Cat,
			PetUptime:      1,
			PetAttackSpeed: 2.0,
		},
	},
}

var ItemFilters = core.ItemFilter{
	ArmorType: proto.ArmorType_ArmorTypeMail,
	WeaponTypes: []proto.WeaponType{
		proto.WeaponType_WeaponTypeAxe,
		proto.WeaponType_WeaponTypeDagger,
		proto.WeaponType_WeaponTypeFist,
		proto.WeaponType_WeaponTypeMace,
		proto.WeaponType_WeaponTypeOffHand,
		proto.WeaponType_WeaponTypePolearm,
		proto.WeaponType_WeaponTypeStaff,
		proto.WeaponType_WeaponTypeSword,
	},
	RangedWeaponTypes: []proto.RangedWeaponType{
		proto.RangedWeaponType_RangedWeaponTypeBow,
		proto.RangedWeaponType_RangedWeaponTypeCrossbow,
		proto.RangedWeaponType_RangedWeaponTypeGun,
	},
}

var Stats = []proto.Stat{
	proto.Stat_StatAgility,
	proto.Stat_StatAttackPower,
	proto.Stat_StatRangedAttackPower,
	proto.Stat_StatMeleeCrit,
	proto.Stat_StatMeleeHit,
}
