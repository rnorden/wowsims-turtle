package hunter

import (
	"time"

	"github.com/isfir/wowsims-turtle/sim/common/guardians"
	"github.com/isfir/wowsims-turtle/sim/core"
	"github.com/isfir/wowsims-turtle/sim/core/proto"
	"github.com/isfir/wowsims-turtle/sim/core/stats"
)

var TalentTreeSizes = [3]int{19, 16, 20}

const (
	SpellFlagShot   = core.SpellFlagAgentReserved1
	SpellFlagStrike = core.SpellFlagAgentReserved2
	SpellFlagSting  = core.SpellFlagAgentReserved3
	SpellFlagTrap   = core.SpellFlagAgentReserved4
)

const (
	SpellCode_HunterNone int32 = iota

	// Shots
	SpellCode_HunterAimedShot
	SpellCode_HunterArcaneShot
	SpellCode_HunterMultiShot
	SpellCode_HunterSteadyShot
	SpellCode_HunterExtraShot

	// Strikes
	SpellCode_HunterRaptorStrike
	SpellCode_HunterRaptorStrikeHit

	// Stings
	SpellCode_HunterSerpentSting

	// Traps
	SpellCode_HunterExplosiveTrap
	SpellCode_HunterFreezingTrap
	SpellCode_HunterImmolationTrap

	// Other
	SpellCode_HunterMongooseBite
	SpellCode_HunterWingClip
	SpellCode_HunterVolley
	SpellCode_HunterCarve
	SpellCode_HunterLacerate
	SpellCode_HunterLacerateBleed
	SpellCode_HunterPiercingShots
	SpellCode_HunterKillCommand

	// Pet Spells
	SpellCode_HunterPetClaw
	SpellCode_HunterPetBite
	SpellCode_HunterPetLightningBreath
	SpellCode_HunterPetScreech
	SpellCode_HunterPetScorpidPoison
)

func RegisterHunter() {
	core.RegisterAgentFactory(
		proto.Player_Hunter{},
		proto.Spec_SpecHunter,
		func(character *core.Character, options *proto.Player) core.Agent {
			return NewHunter(character, options)
		},
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_Hunter)
			if !ok {
				panic("Invalid spec value for Hunter!")
			}
			player.Spec = playerSpec
		},
	)
}

type Hunter struct {
	core.Character

	Talents *proto.HunterTalents
	Options *proto.Hunter_Options

	pet *HunterPet

	AmmoDPS                   float64
	AmmoDamageBonus           float64
	NormalizedAmmoDamageBonus float64

	// Miscellaneous set bonuses that require extra logic inside of spells
	AspectOfTheHawkAPMultiplier float64
	AspectOfTheWolfAPMultiplier float64

	curQueueAura       *core.Aura
	curQueuedAutoSpell *core.Spell

	AimedShot       *core.Spell
	ArcaneShot      *core.Spell
	Carve           *core.Spell
	ExtraShot       *core.Spell
	Lacerate        *core.Spell
	LacerateBleed   *core.Spell
	PiercingShots   *core.Spell
	SteadyShot      *core.Spell
	ExplosiveTrap   *core.Spell
	ImmolationTrap  *core.Spell
	FreezingTrap    *core.Spell
	KillCommand     *core.Spell
	MultiShot       *core.Spell
	RapidFire       *core.Spell
	RaptorStrike    *core.Spell
	RaptorStrikeHit *core.Spell
	MongooseBite    *core.Spell
	MongooseBiteOH  *core.Spell
	ScorpidSting    *core.Spell
	SerpentSting    *core.Spell
	SilencingShot   *core.Spell
	Volley          *core.Spell
	WingClip        *core.Spell

	Shots       []*core.Spell
	Strikes     []*core.Spell
	MeleeSpells []*core.Spell
	LastShot    *core.Spell

	// Set when the hunter lands a critical strike; enables Kill Command and Lacerate.
	KillCommandReadyAura *core.Aura
	LacerateReadyAura    *core.Aura
	LockAndLoadAura      *core.Aura

	RapidFireAura       *core.Aura
	ScentOfBloodPetAura *core.Aura
}

func (hunter *Hunter) GetCharacter() *core.Character {
	return &hunter.Character
}

func (hunter *Hunter) GetHunter() *Hunter {
	return hunter
}

func (hunter *Hunter) AddRaidBuffs(raidBuffs *proto.RaidBuffs) {
}

func (hunter *Hunter) AddPartyBuffs(_ *proto.PartyBuffs) {
}

func (hunter *Hunter) Initialize() {
	hunter.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(SpellFlagShot) {
			hunter.Shots = append(hunter.Shots, spell)
		}
	})
	hunter.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(SpellFlagStrike) {
			hunter.Strikes = append(hunter.Strikes, spell)
		}
	})
	hunter.OnSpellRegistered(func(spell *core.Spell) {
		if spell.ProcMask.Matches(core.ProcMaskMeleeMHSpecial | core.ProcMaskMeleeOHSpecial) {
			hunter.MeleeSpells = append(hunter.MeleeSpells, spell)
		}
	})

	hunter.registerAspectOfTheHawkSpell()
	hunter.registerAspectOfTheWolfSpell()
	hunter.registerAspectOfTheViperSpell()

	// Multi-Shot and Carve share a cooldown.
	multiShotTimer := hunter.NewTimer()

	hunter.registerSerpentStingSpell()

	hunter.registerArcaneShotSpell()
	hunter.registerAimedShotSpell()
	hunter.registerMultiShotSpell(multiShotTimer)
	hunter.registerSteadyShotSpell()
	hunter.registerExtraShotSpell()

	hunter.registerRaptorStrikeSpell()
	hunter.registerMongooseBiteSpell()
	hunter.registerWingClipSpell()
	hunter.registerVolleySpell()
	hunter.registerCarveSpell(multiShotTimer)

	hunter.registerCritReactiveAuras()
	hunter.registerLacerateSpell()
	hunter.registerKillCommandSpell()

	traps := hunter.NewTimer()

	hunter.registerExplosiveTrapSpell(traps)
	hunter.registerImmolationTrapSpell(traps)
	hunter.registerFreezingTrapSpell(traps)

	hunter.registerRapidFire()
}

func (hunter *Hunter) Reset(sim *core.Simulation) {
}

func NewHunter(character *core.Character, options *proto.Player) *Hunter {
	hunterOptions := options.GetHunter()

	hunter := &Hunter{
		Character: *character,
		Talents:   &proto.HunterTalents{},
		Options:   hunterOptions.Options,
	}
	core.FillTalentsProto(hunter.Talents.ProtoReflect(), options.TalentsString, TalentTreeSizes)
	hunter.EnableManaBar()

	hunter.PseudoStats.CanParry = true

	rangedWeapon := hunter.WeaponFromRanged()

	if hunter.HasRangedWeapon() {
		// Ammo
		switch hunter.Options.Ammo {
		case proto.Hunter_Options_RazorArrow:
			hunter.AmmoDPS = 7.5
		case proto.Hunter_Options_SolidShot:
			hunter.AmmoDPS = 7.5
		case proto.Hunter_Options_JaggedArrow:
			hunter.AmmoDPS = 13
		case proto.Hunter_Options_AccurateSlugs:
			hunter.AmmoDPS = 13
		case proto.Hunter_Options_MithrilGyroShot:
			hunter.AmmoDPS = 15
		case proto.Hunter_Options_IceThreadedArrow:
			hunter.AmmoDPS = 16.5
		case proto.Hunter_Options_IceThreadedBullet:
			hunter.AmmoDPS = 16.5
		case proto.Hunter_Options_ThoriumHeadedArrow:
			hunter.AmmoDPS = 17.5
		case proto.Hunter_Options_ThoriumShells:
			hunter.AmmoDPS = 17.5
		case proto.Hunter_Options_RockshardPellets:
			hunter.AmmoDPS = 18
		case proto.Hunter_Options_Doomshot:
			hunter.AmmoDPS = 20
		case proto.Hunter_Options_MiniatureCannonBalls:
			hunter.AmmoDPS = 20.5
		}
		hunter.AmmoDamageBonus = hunter.AmmoDPS * rangedWeapon.SwingSpeed
		hunter.NormalizedAmmoDamageBonus = hunter.AmmoDPS * 2.8

		// Quiver
		switch hunter.Options.QuiverBonus {
		case proto.Hunter_Options_Speed10:
			hunter.PseudoStats.RangedSpeedMultiplier *= 1.1
		case proto.Hunter_Options_Speed11:
			hunter.PseudoStats.RangedSpeedMultiplier *= 1.11
		case proto.Hunter_Options_Speed12:
			hunter.PseudoStats.RangedSpeedMultiplier *= 1.12
		case proto.Hunter_Options_Speed13:
			hunter.PseudoStats.RangedSpeedMultiplier *= 1.13
		case proto.Hunter_Options_Speed14:
			hunter.PseudoStats.RangedSpeedMultiplier *= 1.14
		case proto.Hunter_Options_Speed15:
			hunter.PseudoStats.RangedSpeedMultiplier *= 1.15
		}
	}

	hunter.EnableAutoAttacks(hunter, core.AutoAttackOptions{
		MainHand:        hunter.WeaponFromMainHand(),
		OffHand:         hunter.WeaponFromOffHand(),
		Ranged:          rangedWeapon,
		ReplaceMHSwing:  hunter.TryRaptorStrike,
		AutoSwingRanged: true,
		AutoSwingMelee:  true,
	})

	hunter.AutoAttacks.RangedConfig().Flags |= core.SpellFlagCastTimeNoGCD
	hunter.AutoAttacks.RangedConfig().Cast = core.CastConfig{
		DefaultCast: core.Cast{
			CastTime: time.Millisecond * 500,
		},
		ModifyCast: func(_ *core.Simulation, spell *core.Spell, cast *core.Cast) {
			cast.CastTime = spell.CastTime()
		},
		IgnoreHaste: true, // Hunter GCD is locked at 1.5s
		CastTime: func(spell *core.Spell) time.Duration {
			return time.Duration(float64(spell.DefaultCast.CastTime) / hunter.RangedSwingSpeed())
		},
	}
	hunter.AutoAttacks.RangedConfig().ExtraCastCondition = func(sim *core.Simulation, target *core.Unit) bool {
		return !hunter.IsCasting(sim)
	}
	hunter.AutoAttacks.RangedConfig().CritDamageBonus = hunter.mortalShots()
	hunter.AutoAttacks.RangedConfig().BonusCoefficient = 1
	hunter.AutoAttacks.RangedConfig().ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		baseDamage := hunter.RangedWeaponDamage(sim, spell.RangedAttackPower(target, false)) +
			hunter.AmmoDamageBonus
		result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeRangedHitAndCrit)
		hunter.AutoAttacks.AddWeaponBonusDamage(sim, hunter.AutoAttacks.Ranged(), spell, result)

		spell.WaitTravelTime(sim, func(sim *core.Simulation) {
			spell.DealDamage(sim, result)
		})
	}

	hunter.pet = hunter.NewHunterPet()

	hunter.AddStatDependency(stats.Strength, stats.AttackPower, core.APPerStrength[character.Class])
	hunter.AddStatDependency(stats.Agility, stats.AttackPower, 1)
	hunter.AddStatDependency(stats.Agility, stats.RangedAttackPower, 2)
	hunter.AddStatDependency(stats.Agility, stats.MeleeCrit, core.CritPerAgiAtLevel[character.Class]*core.CritRatingPerCritChance)
	hunter.AddStatDependency(stats.Intellect, stats.SpellCrit, core.CritPerIntAtLevel[character.Class]*core.SpellCritRatingPerCritChance)

	guardians.ConstructGuardians(&hunter.Character)

	return hunter
}

func (hunter *Hunter) OnGCDReady(_ *core.Simulation) {
}

// RangedCastWindup models the hidden auto-shot style windup of hunter ranged casts.
// The client cast times of Aimed Shot, Multi-Shot and Steady Shot do not include it.
const RangedCastWindup = time.Millisecond * 500

// shotCastTime returns the cast time of a hunter shot. In Turtle WoW, ranged haste does not
// shorten shot casts; Rapid Fire reduces them explicitly through CastTimeMultiplier.
func (hunter *Hunter) shotCastTime(spell *core.Spell) time.Duration {
	return time.Duration(float64(spell.DefaultCast.CastTime)*spell.CastTimeMultiplier) + RangedCastWindup
}

// rangedWeaponDamage returns ranged weapon damage including ammo, using the current ranged attack power.
func (hunter *Hunter) rangedWeaponDamage(sim *core.Simulation, spell *core.Spell, target *core.Unit, normalized bool) float64 {
	rap := spell.RangedAttackPower(target, false)
	if normalized {
		return hunter.AutoAttacks.Ranged().CalculateNormalizedWeaponDamage(sim, rap) + hunter.AmmoDamageBonus
	}
	return hunter.RangedWeaponDamage(sim, rap) + hunter.AmmoDamageBonus
}

// Agent is a generic way to access underlying hunter on any of the agents.
type HunterAgent interface {
	GetHunter() *Hunter
}
