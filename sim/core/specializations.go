package core

///////////////////////////////////////////////////////////////////////////
//                            Weapon Specialization Auras
///////////////////////////////////////////////////////////////////////////

// Turtle WoW patch 1.17.2: "Reduced the amount of weapon skill granted by racials from +5 to +3."
const RacialWeaponSkillBonus = 3

func (character *Character) SwordSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Sword Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.SwordsSkill += RacialWeaponSkillBonus
			character.PseudoStats.TwoHandedSwordsSkill += RacialWeaponSkillBonus
		},
	})
}

func (character *Character) AxeSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Axe Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.AxesSkill += RacialWeaponSkillBonus
			character.PseudoStats.TwoHandedAxesSkill += RacialWeaponSkillBonus
		},
	})
}

func (character *Character) MaceSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Mace Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.MacesSkill += RacialWeaponSkillBonus
			character.PseudoStats.TwoHandedMacesSkill += RacialWeaponSkillBonus
		},
	})
}

func (character *Character) DaggerSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Dagger Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.DaggersSkill += RacialWeaponSkillBonus
		},
	})
}

func (character *Character) FistWeaponSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Fist Weapon Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.UnarmedSkill += RacialWeaponSkillBonus
		},
	})
}

func (character *Character) PoleWeaponSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Pole Weapon Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.StavesSkill += RacialWeaponSkillBonus
			character.PseudoStats.PolearmsSkill += RacialWeaponSkillBonus
		},
	})
}

func (character *Character) GunSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Gun Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.GunsSkill += RacialWeaponSkillBonus
		},
	})
}

func (character *Character) BowSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Bow Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.BowsSkill += RacialWeaponSkillBonus
		},
	})
}

func (character *Character) CrossbowSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Crossbow Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.CrossbowsSkill += RacialWeaponSkillBonus
		},
	})
}

func (character *Character) ThrownSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Thrown Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.ThrownSkill += RacialWeaponSkillBonus
		},
	})
}

func (character *Character) FeralCombatSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Feral Combat Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.FeralCombatSkill += RacialWeaponSkillBonus
		},
	})
}
