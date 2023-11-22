package constant

import "hk4e/pkg/endec"

var (
	DEFAULT_ABILITY_HASH_CODE      []int32
	DEFAULT_TEAM_ABILITY_HASH_CODE []int32
)

func init() {
	abilityList := []string{
		"Avatar_DefaultAbility_VisionReplaceDieInvincible",
		"Avatar_DefaultAbility_AvartarInShaderChange",
		"Avatar_SprintBS_Invincible",
		"Avatar_Freeze_Duration_Reducer",
		"Avatar_Attack_ReviveEnergy",
		"Avatar_Component_Initializer",
		"Avatar_FallAnthem_Achievement_Listener",
		"GrapplingHookSkill_Ability",
		"Avatar_PlayerBoy_DiveStamina_Reduction",
		"Ability_Avatar_Dive_SealEcho",
		"Absorb_SealEcho_Bullet_01",
		"Absorb_SealEcho_Bullet_02",
		"Ability_Avatar_Dive_CrabShield",
		"ActivityAbility_Absorb_Shoot",
		"SceneAbility_DiveVolume",
	}
	DEFAULT_ABILITY_HASH_CODE = make([]int32, 0)
	for _, ability := range abilityList {
		DEFAULT_ABILITY_HASH_CODE = append(DEFAULT_ABILITY_HASH_CODE, endec.Hk4eAbilityHashCode(ability))
	}
	abilityList = []string{
		"Ability_Avatar_Dive_Team",
	}
	DEFAULT_TEAM_ABILITY_HASH_CODE = make([]int32, 0)
	for _, ability := range abilityList {
		DEFAULT_TEAM_ABILITY_HASH_CODE = append(DEFAULT_TEAM_ABILITY_HASH_CODE, endec.Hk4eAbilityHashCode(ability))
	}
}
