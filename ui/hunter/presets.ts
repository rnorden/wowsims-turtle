import { Phase } from '../core/constants/other.js';
import * as PresetUtils from '../core/preset_utils.js';
import {
	AgilityElixir,
	Alcohol,
	AttackPowerBuff,
	Conjured,
	Consumes,
	Debuffs,
	Flask,
	Food,
	HealthElixir,
	IndividualBuffs,
	Potions,
	Profession,
	Race,
	RaidBuffs,
	SapperExplosive,
	SaygesFortune,
	SpellPowerBuff,
	StrengthBuff,
	TristateEffect,
	WeaponImbue,
	ZanzaBuff,
} from '../core/proto/common.js';
import {
	Hunter_Options as HunterOptions,
	Hunter_Options_Ammo as Ammo,
	Hunter_Options_PetAttackSpeed as PetAttackSpeed,
	Hunter_Options_PetType as PetType,
	Hunter_Options_QuiverBonus,
} from '../core/proto/hunter.js';
import { SavedTalents } from '../core/proto/ui.js';
import BMAPL from './apls/bm.apl.json';
import BMNoBuffsAPL from './apls/bm_nobuffs.apl.json';
import P1APL from './apls/p1.apl.json';
import BMAlt01Gear from './gear_sets/bm_alt01.gear.json';
import BMAlt02Gear from './gear_sets/bm_alt02.gear.json';
import BMAlt03Gear from './gear_sets/bm_alt03.gear.json';
import BMAlt04Gear from './gear_sets/bm_alt04.gear.json';
import BMAlt05Gear from './gear_sets/bm_alt05.gear.json';
import BMAlt06Gear from './gear_sets/bm_alt06.gear.json';
import BMAlt07Gear from './gear_sets/bm_alt07.gear.json';
import BMAlt08Gear from './gear_sets/bm_alt08.gear.json';
import BMAlt09Gear from './gear_sets/bm_alt09.gear.json';
import BMAlt10Gear from './gear_sets/bm_alt10.gear.json';
import BMNoBuffsGear from './gear_sets/bm_nobuffs.gear.json';
import BMPreBISGear from './gear_sets/bm_prebis.gear.json';
import BMPreBISNoT05ExaltedGear from './gear_sets/bm_prebis_no_t05_exalted.gear.json';
import CurrentEadiGear from './gear_sets/current-eadi.gear.json';
import P0BISGear from './gear_sets/p0.bis.gear.json';
import P1BISGear from './gear_sets/p1.bis.gear.json';

// Preset options for this spec.
// Eventually we will import these values for the raid sim too, so its good to
// keep them in a separate file.
///////////////////////////////////////////////////////////////////////////
//                                 Gear Presets
///////////////////////////////////////////////////////////////////////////

// Derived with the sim: best simulated BM DPS from non-raid Turtle WoW items (Horde, pre-raid sources).
export const GearBMPreBIS = PresetUtils.makePresetGear('BM Pre-BiS', BMPreBISGear);
// Same method, without Tier 0.5 (dungeon set 2) pieces and exalted reputation rewards.
export const GearBMPreBISNoT05Exalted = PresetUtils.makePresetGear('BM Pre-BiS (no T0.5/exalted)', BMPreBISNoT05ExaltedGear);
// Best of the BM pre-raid alternatives without buffs/consumables (slow bow; see the no-buffs rotation).
export const GearBMNoBuffs = PresetUtils.makePresetGear('BM no-buffs', BMNoBuffsGear);
// The 10 best sim-derived BM pre-raid alternatives (no T0.5/exalted pool), ranked by DPS with buffs.
export const GearBMAlt01 = PresetUtils.makePresetGear('BM alt #1', BMAlt01Gear);
export const GearBMAlt02 = PresetUtils.makePresetGear('BM alt #2', BMAlt02Gear);
export const GearBMAlt03 = PresetUtils.makePresetGear('BM alt #3', BMAlt03Gear);
export const GearBMAlt04 = PresetUtils.makePresetGear('BM alt #4', BMAlt04Gear);
export const GearBMAlt05 = PresetUtils.makePresetGear('BM alt #5', BMAlt05Gear);
export const GearBMAlt06 = PresetUtils.makePresetGear('BM alt #6', BMAlt06Gear);
export const GearBMAlt07 = PresetUtils.makePresetGear('BM alt #7', BMAlt07Gear);
export const GearBMAlt08 = PresetUtils.makePresetGear('BM alt #8', BMAlt08Gear);
export const GearBMAlt09 = PresetUtils.makePresetGear('BM alt #9', BMAlt09Gear);
export const GearBMAlt10 = PresetUtils.makePresetGear('BM alt #10', BMAlt10Gear);
export const GearCurrentEadi = PresetUtils.makePresetGear('current-eadi', CurrentEadiGear);
export const GearP0BIS = PresetUtils.makePresetGear('Pre-BiS', P0BISGear);
export const GearP1BIS = PresetUtils.makePresetGear('P1 BiS', P1BISGear);

export const GearPresets = {
	[Phase.Phase1]: [
		GearBMPreBIS,
		GearBMPreBISNoT05Exalted,
		GearBMNoBuffs,
		GearBMAlt01,
		GearBMAlt02,
		GearBMAlt03,
		GearBMAlt04,
		GearBMAlt05,
		GearBMAlt06,
		GearBMAlt07,
		GearBMAlt08,
		GearBMAlt09,
		GearBMAlt10,
		GearCurrentEadi,
		GearP0BIS,
		GearP1BIS,
	],
};

export const DefaultGear = GearBMPreBIS;

///////////////////////////////////////////////////////////////////////////
//                                 APL Presets
///////////////////////////////////////////////////////////////////////////

// Buffed: full shot priority, expensive shots only above 10% mana.
export const APLBM = PresetUtils.makePresetAPLRotation('Beast Mastery (buffed)', BMAPL);
// No buffs: mana-starved, so no Serpent Sting and Aimed/Multi/Arcane only above 60% mana; Steady Shot fills.
export const APLBMNoBuffs = PresetUtils.makePresetAPLRotation('Beast Mastery (no buffs)', BMNoBuffsAPL);
export const APLP1 = PresetUtils.makePresetAPLRotation('Marksmanship', P1APL);

export const APLPresets = {
	[Phase.Phase1]: [APLBM, APLBMNoBuffs, APLP1],
};

export const DefaultAPL = APLPresets[Phase.Phase1][0];

///////////////////////////////////////////////////////////////////////////
//                                 Talent Presets
///////////////////////////////////////////////////////////////////////////

// Default talents. Uses the talent string format (numeric), make the talents on
// a talent calculator and copy the numbers in the url.

// Turtle WoW talent trees (BM 19, MM 16, Survival 20 talents).
export const TalentsBM = PresetUtils.makePresetTalents('Beast Mastery', SavedTalents.create({ talentsString: '5500000152053102251-0505013' }));
export const TalentsP1 = PresetUtils.makePresetTalents('Marksmanship', SavedTalents.create({ talentsString: '550000005-0505113250123251' }));

export const TalentPresets = {
	[Phase.Phase1]: [TalentsBM, TalentsP1],
};

export const DefaultTalents = TalentPresets[Phase.Phase1][0];

///////////////////////////////////////////////////////////////////////////
//                                 Options
///////////////////////////////////////////////////////////////////////////

export const DefaultOptions = HunterOptions.create({
	ammo: Ammo.ThoriumHeadedArrow,
	quiverBonus: Hunter_Options_QuiverBonus.Speed15,
	petAttackSpeed: PetAttackSpeed.OneTwo,
	petType: PetType.Cat,
	petUptime: 1,
});

export const DefaultConsumes = Consumes.create({
	agilityElixir: AgilityElixir.ElixirOfTheMongoose,
	alcohol: Alcohol.AlcoholRumseyRumBlackLabel,
	attackPowerBuff: AttackPowerBuff.JujuMight,
	defaultConjured: Conjured.ConjuredDemonicRune,
	defaultPotion: Potions.MajorManaPotion,
	flask: Flask.FlaskOfSupremePower,
	food: Food.FoodSmokedDesertDumpling,
	healthElixir: HealthElixir.ElixirOfFortitude,
	mainHandImbue: WeaponImbue.Windfury,
	offHandImbue: WeaponImbue.ElementalSharpeningStone,
	petAttackPowerConsumable: 1,
	petAgilityConsumable: 1,
	petStrengthConsumable: 1,
	sapperExplosive: SapperExplosive.SapperUnknown,
	spellPowerBuff: SpellPowerBuff.GreaterArcaneElixir,
	strengthBuff: StrengthBuff.JujuPower,
});

export const DefaultRaidBuffs = RaidBuffs.create({
	arcaneBrilliance: true,
	battleShout: TristateEffect.TristateEffectImproved,
	divineSpirit: true,
	fireResistanceAura: true,
	fireResistanceTotem: true,
	giftOfTheWild: TristateEffect.TristateEffectImproved,
	graceOfAirTotem: TristateEffect.TristateEffectImproved,
	leaderOfThePack: false,
	manaSpringTotem: TristateEffect.TristateEffectRegular,
	strengthOfEarthTotem: TristateEffect.TristateEffectImproved,
});

export const DefaultIndividualBuffs = IndividualBuffs.create({
	blessingOfKings: true,
	blessingOfMight: TristateEffect.TristateEffectImproved,
	blessingOfWisdom: TristateEffect.TristateEffectImproved,
	fengusFerocity: false,
	moldarsMoxie: false,
	rallyingCryOfTheDragonslayer: true,
	saygesFortune: SaygesFortune.SaygesDamage,
	slipkiksSavvy: false,
	songflowerSerenade: true,
	spiritOfZandalar: false,
	warchiefsBlessing: false,
});

export const DefaultDebuffs = Debuffs.create({
	curseOfRecklessness: true,
	exposeArmor: TristateEffect.TristateEffectImproved,
	faerieFire: true,
	huntersMark: TristateEffect.TristateEffectImproved,
	fireVulnerability: true,
	judgementOfWisdom: true,
	stormstrike: false,
	sunderArmor: true,
});

// No raid/party/player buffs or consumables; only the hunter's own Hunter's Mark.
export const SettingsNoBuffs = PresetUtils.makePresetEncounter('no-buffs', undefined, {
	raidBuffs: RaidBuffs.create(),
	buffs: IndividualBuffs.create(),
	debuffs: Debuffs.create({ huntersMark: TristateEffect.TristateEffectRegular }),
	consumes: Consumes.create(),
});

export const BuildNoBuffs = PresetUtils.makePresetBuild('no-buffs', { encounter: SettingsNoBuffs, rotation: APLBMNoBuffs, gear: GearBMNoBuffs });

// Typical raid group buffs and personal consumables, without world buffs.
export const SettingsRaidBuffs = PresetUtils.makePresetEncounter('raid-buffs', undefined, {
	raidBuffs: RaidBuffs.create({
		arcaneBrilliance: true,
		battleShout: TristateEffect.TristateEffectRegular,
		divineSpirit: true,
		giftOfTheWild: TristateEffect.TristateEffectImproved,
		graceOfAirTotem: TristateEffect.TristateEffectImproved,
		leaderOfThePack: true,
		powerWordFortitude: TristateEffect.TristateEffectImproved,
		strengthOfEarthTotem: TristateEffect.TristateEffectImproved,
	}),
	buffs: IndividualBuffs.create(),
	debuffs: Debuffs.create({ huntersMark: TristateEffect.TristateEffectRegular }),
	consumes: Consumes.create({
		agilityElixir: AgilityElixir.ElixirOfTheMongoose,
		attackPowerBuff: AttackPowerBuff.JujuMight,
		defaultConjured: Conjured.ConjuredDemonicRune,
		defaultPotion: Potions.MajorManaPotion,
		flask: Flask.FlaskOfTheTitans,
		food: Food.FoodGrilledSquid,
		petAttackPowerConsumable: 1,
		zanzaBuff: ZanzaBuff.SpiritOfZanza,
	}),
});

export const BuildRaidBuffs = PresetUtils.makePresetBuild('raid-buffs', { encounter: SettingsRaidBuffs, rotation: APLBM });

export const OtherDefaults = {
	distanceFromTarget: 12,
	profession1: Profession.Enchanting,
	profession2: Profession.Engineering,
	race: Race.RaceTroll,
};
