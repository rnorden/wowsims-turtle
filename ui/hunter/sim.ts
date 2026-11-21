import * as BuffDebuffInputs from '../core/components/inputs/buffs_debuffs';
import * as ConsumablesInputs from '../core/components/inputs/consumables.js';
import * as OtherInputs from '../core/components/other_inputs.js';
import { Phase } from '../core/constants/other.js';
import { IndividualSimUI, registerSpecConfig } from '../core/individual_sim_ui.js';
import { Player } from '../core/player.js';
import { PartyBuffs, PseudoStat, Spec, Stat } from '../core/proto/common.js';
import { Stats } from '../core/proto_utils/stats.js';
import * as HunterInputs from './inputs.js';
import * as Presets from './presets.js';

const SPEC_CONFIG = registerSpecConfig(Spec.SpecHunter, {
	cssClass: 'hunter-sim-ui',
	cssScheme: 'hunter',
	// List any known bugs / issues here and they'll be shown on the site.
	knownIssues: [],
	warnings: [],

	// All stats for which EP should be calculated.
	epStats: [
		// Attributes
		Stat.StatIntellect,
		Stat.StatStrength,
		Stat.StatAgility,
		// Physical
		Stat.StatAttackPower,
		Stat.StatRangedAttackPower,
		Stat.StatMeleeHit,
		Stat.StatMeleeCrit,
		Stat.StatExpertise,
		// Spell
		Stat.StatSpellPower,
		Stat.StatSpellDamage,
		Stat.StatNaturePower,
		Stat.StatArcanePower,
		Stat.StatSpellCrit,
		Stat.StatMP5,
	],
	epPseudoStats: [
		PseudoStat.PseudoStatMainHandDps,
		PseudoStat.PseudoStatOffHandDps,
		PseudoStat.PseudoStatRangedDps,
		PseudoStat.PseudoStatMeleeSpeedMultiplier,
		PseudoStat.PseudoStatRangedSpeedMultiplier,
		PseudoStat.PseudoStatBowsSkill,
		PseudoStat.PseudoStatCrossbowsSkill,
		PseudoStat.PseudoStatGunsSkill,
	],
	// Reference stat against which to calculate EP.
	epReferenceStat: Stat.StatRangedAttackPower,
	// Which stats to display in the Character Stats section, at the bottom of the left-hand sidebar.
	displayStats: [
		// Primary
		Stat.StatMana,
		// Attributes
		Stat.StatStrength,
		Stat.StatAgility,
		Stat.StatIntellect,
		// Physical
		Stat.StatAttackPower,
		Stat.StatRangedAttackPower,
		Stat.StatMeleeHit,
		Stat.StatMeleeCrit,
		Stat.StatExpertise,
		// Spell
		Stat.StatSpellDamage,
		Stat.StatNaturePower,
		Stat.StatArcanePower,
		Stat.StatSpellCrit,
		Stat.StatMP5,
	],
	displayPseudoStats: [PseudoStat.PseudoStatRangedHit, PseudoStat.PseudoStatRangedCrit, PseudoStat.PseudoStatMeleeSpeedMultiplier, PseudoStat.PseudoStatRangedSpeedMultiplier],

	defaults: {
		race: Presets.OtherDefaults.race,
		// Default equipped gear.
		gear: Presets.DefaultGear.gear,
		// Default EP weights for sorting gear in the gear picker.
		// Measured 2026-11-21 (OctoWoW rules) with the UI defaults: Troll, Enchanting/Engineering, 12 yd, BM talents, BM (buffed)
		// APL, Thorium Headed Arrow, 15% quiver, Cat at 1.2 attack speed, default consumes/raid/individual buffs/debuffs, no party
		// buffs, 120 s. Paired-seed DPS deltas at 20k iterations, averaged over the BM Pre-BiS and current-eadi sets.
		// Melee AP, Strength, Stamina and melee weapon stats are worth nothing to a ranged BM rotation (the pet does not scale with them).
		// Hit is the below-cap value (BM Pre-BiS sits at the 8% cap, so extra hit there is worth ~0).
		// Ranged speed is the slope of a linear fit over -5%..+5%: auto-shot timing locks onto the GCD grid, so single-point values swing.
		// Weapon skill: 5 skill = 1% hit. Only Guns (the default gear's weapon) is weighted, so gear with +bows/crossbows/guns skill
		// counts once; users of a bow or crossbow should run stat weights.
		epWeights: Stats.fromMap(
			{
				[Stat.StatStrength]: 0,
				[Stat.StatAgility]: 2.41,
				[Stat.StatStamina]: 0,
				[Stat.StatIntellect]: 0.16,
				[Stat.StatAttackPower]: 0,
				[Stat.StatRangedAttackPower]: 1.0,
				[Stat.StatMeleeHit]: 25.14,
				[Stat.StatMeleeCrit]: 21.57,
				[Stat.StatSpellPower]: 0.16,
				[Stat.StatNaturePower]: 0.16,
				[Stat.StatArcanePower]: 0,
				[Stat.StatSpellCrit]: 0,
				[Stat.StatMP5]: 0.26,
			},
			{
				[PseudoStat.PseudoStatMainHandDps]: 0,
				[PseudoStat.PseudoStatOffHandDps]: 0,
				[PseudoStat.PseudoStatRangedDps]: 8.86,
				[PseudoStat.PseudoStatMeleeSpeedMultiplier]: 0,
				[PseudoStat.PseudoStatRangedSpeedMultiplier]: 9.67,
				[PseudoStat.PseudoStatGunsSkill]: 5.03,
			},
		),
		// Default consumes settings.
		consumes: Presets.DefaultConsumes,
		// Default talents.
		talents: Presets.DefaultTalents.data,
		// Default spec-specific settings.
		specOptions: Presets.DefaultOptions,
		other: Presets.OtherDefaults,
		// Default raid/party buffs settings.
		raidBuffs: Presets.DefaultRaidBuffs,
		partyBuffs: PartyBuffs.create({}),
		individualBuffs: Presets.DefaultIndividualBuffs,
		debuffs: Presets.DefaultDebuffs,
	},

	// IconInputs to include in the 'Player' section on the settings tab.
	playerIconInputs: [HunterInputs.PetTypeInput, HunterInputs.WeaponAmmo, HunterInputs.QuiverInput],
	// Inputs to include in the 'Rotation' section on the settings tab.
	rotationInputs: HunterInputs.HunterRotationConfig,
	petConsumeInputs: [ConsumablesInputs.PetAttackPowerConsumable, ConsumablesInputs.PetAgilityConsumable, ConsumablesInputs.PetStrengthConsumable],
	// Buff and Debuff inputs to include/exclude, overriding the EP-based defaults.
	includeBuffDebuffInputs: [
		ConsumablesInputs.DragonBreathChili,
		BuffDebuffInputs.SpellFireVulnerabilityDebuff,
		BuffDebuffInputs.StaminaBuff,
		BuffDebuffInputs.SpiritBuff,
	],
	excludeBuffDebuffInputs: [],
	// Inputs to include in the 'Other' section on the settings tab.
	otherInputs: {
		inputs: [
			//HunterInputs.NewRaptorStrike,
			HunterInputs.PetAttackSpeedInput,
			HunterInputs.PetUptime,
			OtherInputs.DistanceFromTarget,
			OtherInputs.TankAssignment,
			OtherInputs.InFrontOfTarget,
		],
	},
	encounterPicker: {
		// Whether to include 'Execute Duration (%)' in the 'Encounter' section of the settings tab.
		showExecuteProportion: false,
	},

	presets: {
		// Preset talents that the user can quickly select.
		talents: [...Presets.TalentPresets[Phase.Phase1]],
		// Preset rotations that the user can quickly select.
		rotations: [...Presets.APLPresets[Phase.Phase1]],
		// Preset gear configurations that the user can quickly select.
		gear: [...Presets.GearPresets[Phase.Phase1]],
		// Preset configurations (shown on the settings tab for buff/consume-only presets).
		builds: [Presets.BuildNoBuffs, Presets.BuildRaidBuffs],
	},

	autoRotation: player => {
		const isMelee = false;
		//player.hasRune(ItemSlot.ItemSlotWaist, HunterRune.RuneBeltMeleeSpecialist) ||
		//player.hasRune(ItemSlot.ItemSlotFeet, HunterRune.RuneBootsDualWieldSpecialization) ||
		//player.hasRune(ItemSlot.ItemSlotFeet, HunterRune.RuneBootsWyvernStrike);

		return Presets.DefaultAPL.rotation.rotation!;

		// COMMENTING OUT TO SAVE FOR FUTURE IMPLEMENTATION.
		// if (isMelee) {
		// 	switch (level) {
		// 		case 25:
		// 			return Presets.APLMeleeWeavePhase1.rotation.rotation!;
		// 		case 40:
		// 			return Presets.APLMeleePhase2.rotation.rotation!;
		// 		case 50:
		// 			return Presets.APLMeleeBmPhase3.rotation.rotation!;
		// 		case 60:
		// 			return Presets.APLWeavePhase4.rotation.rotation!;
		// 	}
		// } else {
		// 	switch (level) {
		// 		case 25:
		// 			return Presets.APLMeleeWeavePhase1.rotation.rotation!;
		// 		case 40:
		// 			return player.getTalentTree() === 1 ? Presets.APLRangedMmPhase2.rotation.rotation! : Presets.APLRangedBmPhase2.rotation.rotation!;
		// 		case 50:
		// 			return Presets.APLRangedMmPhase3.rotation.rotation!;
		// 		case 60:
		// 			return Presets.APLRangedPhase4.rotation.rotation!;
		// 	}
		// }
		//throw new Error('Auto rotation not supported for your current configuration.');
	},

	raidSimPresets: [
		// Raid sim presets dont work very well with SoD specs between phases
		// and we dont support raid sim atm so just comment this out
		// {
		// 	spec: Spec.SpecHunter,
		// 	tooltip: 'Beast Mastery Hunter',
		// 	defaultName: 'Beast Mastery',
		// 	iconUrl: getSpecIcon(Class.ClassHunter, 0),
		// 	talents: Presets.DefaultTalentsBeastMastery.data,
		// 	specOptions: Presets.BMDefaultOptions,
		// 	consumes: Presets.DefaultConsumes,
		// 	defaultFactionRaces: {
		// 		[Faction.Unknown]: Race.RaceUnknown,
		// 		[Faction.Alliance]: Race.RaceNightElf,
		// 		[Faction.Horde]: Race.RaceOrc,
		// 	},
		// 	defaultGear: {
		// 		[Faction.Unknown]: {},
		// 		[Faction.Alliance]: {
		// 			1: Presets.GearPresets[Phase.Phase1][0].gear,
		// 		},
		// 		[Faction.Horde]: {
		// 			1: Presets.GearPresets[Phase.Phase1][0].gear,
		// 		},
		// 	},
		// },
		// {
		// 	spec: Spec.SpecHunter,
		// 	tooltip: 'Marksmanship Hunter',
		// 	defaultName: 'Marksmanship',
		// 	iconUrl: getSpecIcon(Class.ClassHunter, 1),
		// 	talents: Presets.DefaultTalentsMarksman.data,
		// 	specOptions: Presets.DefaultOptions,
		// 	consumes: Presets.DefaultConsumes,
		// 	defaultFactionRaces: {
		// 		[Faction.Unknown]: Race.RaceUnknown,
		// 		[Faction.Alliance]: Race.RaceNightElf,
		// 		[Faction.Horde]: Race.RaceOrc,
		// 	},
		// 	defaultGear: {
		// 		[Faction.Unknown]: {},
		// 		[Faction.Alliance]: {
		// 			1: Presets.GearPresets[Phase.Phase1][1].gear,
		// 		},
		// 		[Faction.Horde]: {
		// 			1: Presets.GearPresets[Phase.Phase1][1].gear,
		// 		},
		// 	},
		// },
		// {
		// 	spec: Spec.SpecHunter,
		// 	tooltip: 'Survival Hunter',
		// 	defaultName: 'Survival',
		// 	iconUrl: getSpecIcon(Class.ClassHunter, 2),
		// 	talents: Presets.DefaultTalentsSurvival.data,
		// 	specOptions: Presets.DefaultOptions,
		// 	consumes: Presets.DefaultConsumes,
		// 	defaultFactionRaces: {
		// 		[Faction.Unknown]: Race.RaceUnknown,
		// 		[Faction.Alliance]: Race.RaceNightElf,
		// 		[Faction.Horde]: Race.RaceOrc,
		// 	},
		// 	defaultGear: {
		// 		[Faction.Unknown]: {},
		// 		[Faction.Alliance]: {
		// 			1: Presets.GearPresets[Phase.Phase1][2].gear,
		// 		},
		// 		[Faction.Horde]: {
		// 			1: Presets.GearPresets[Phase.Phase1][2].gear,
		// 		},
		// 	},
		// },
	],
});

export class HunterSimUI extends IndividualSimUI<Spec.SpecHunter> {
	constructor(parentElem: HTMLElement, player: Player<Spec.SpecHunter>) {
		super(parentElem, player, SPEC_CONFIG);
	}
}
