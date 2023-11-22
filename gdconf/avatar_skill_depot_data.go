package gdconf

import (
	"fmt"
	"os"

	"hk4e/pkg/endec"
	"hk4e/pkg/logger"

	"github.com/hjson/hjson-go/v4"
)

// AvatarSkillDepotData 角色技能库配置表
type AvatarSkillDepotData struct {
	AvatarSkillDepotId int32 `csv:"ID"`
	// 元素爆发
	EnergySkill int32 `csv:"充能技能,omitempty"`
	// 其他战斗天赋
	Skill1 int32 `csv:"技能1,omitempty"`
	Skill2 int32 `csv:"技能2,omitempty"`
	Skill3 int32 `csv:"技能3,omitempty"`
	Skill4 int32 `csv:"技能4,omitempty"`
	// 命座
	Talent1 int32 `csv:"天赋1,omitempty"`
	Talent2 int32 `csv:"天赋2,omitempty"`
	Talent3 int32 `csv:"天赋3,omitempty"`
	Talent4 int32 `csv:"天赋4,omitempty"`
	Talent5 int32 `csv:"天赋5,omitempty"`
	Talent6 int32 `csv:"天赋6,omitempty"`
	// 固有天赋
	ProudSkill1GroupId                int32  `csv:"固有得意技组1ID,omitempty"`
	ProudSkill1NeedAvatarPromoteLevel int32  `csv:"固有得意技组1激活所需角色突破等级,omitempty"`
	ProudSkill2GroupId                int32  `csv:"固有得意技组2ID,omitempty"`
	ProudSkill2NeedAvatarPromoteLevel int32  `csv:"固有得意技组2激活所需角色突破等级,omitempty"`
	ProudSkill3GroupId                int32  `csv:"固有得意技组3ID,omitempty"`
	ProudSkill3NeedAvatarPromoteLevel int32  `csv:"固有得意技组3激活所需角色突破等级,omitempty"`
	ProudSkill4GroupId                int32  `csv:"固有得意技组4ID,omitempty"`
	ProudSkill4NeedAvatarPromoteLevel int32  `csv:"固有得意技组4激活所需角色突破等级,omitempty"`
	ProudSkill5GroupId                int32  `csv:"固有得意技组5ID,omitempty"`
	ProudSkill5NeedAvatarPromoteLevel int32  `csv:"固有得意技组5激活所需角色突破等级,omitempty"`
	SkillDepotAbilityGroup            string `csv:"AbilityGroup,omitempty"`

	Skills                  []int32                    // 其他战斗天赋
	Talents                 []int32                    // 命座
	InherentProudSkillOpens []*InherentProudSkillOpens // 固有天赋
	AbilityHashCodeList     []int32
}

type InherentProudSkillOpens struct {
	ProudSkillGroupId      int32 `json:"proudSkillGroupId"`      // 固有得意技组ID
	NeedAvatarPromoteLevel int32 `json:"needAvatarPromoteLevel"` // 固有得意技组激活所需角色突破等级
}

func (g *GameDataConfig) loadAvatarSkillDepotData() {
	g.AvatarSkillDepotDataMap = make(map[int32]*AvatarSkillDepotData)
	avatarSkillDepotDataList := make([]*AvatarSkillDepotData, 0)
	readTable[AvatarSkillDepotData](g.txtPrefix+"AvatarSkillDepotData.txt", &avatarSkillDepotDataList)
	playerElementsFilePath := g.jsonPrefix + "ability_group/AbilityGroup_Other_PlayerElementAbility.json"
	playerElementsFile, err := os.ReadFile(playerElementsFilePath)
	if err != nil {
		info := fmt.Sprintf("open file error: %v", err)
		panic(info)
	}
	playerAbilities := make(map[string]*ConfigAvatar)
	err = hjson.Unmarshal(playerElementsFile, &playerAbilities)
	if err != nil {
		info := fmt.Sprintf("parse file error: %v", err)
		panic(info)
	}
	logger.Info("load %v PlayerAbilities", len(playerAbilities))

	for _, avatarSkillDepotData := range avatarSkillDepotDataList {
		// 把全部技能数据放进一个列表里 以后要是没用到或者不需要的话就可以删了
		avatarSkillDepotData.Skills = make([]int32, 0)
		if avatarSkillDepotData.Skill1 != 0 {
			avatarSkillDepotData.Skills = append(avatarSkillDepotData.Skills, avatarSkillDepotData.Skill1)
		}
		if avatarSkillDepotData.Skill2 != 0 {
			avatarSkillDepotData.Skills = append(avatarSkillDepotData.Skills, avatarSkillDepotData.Skill2)
		}
		if avatarSkillDepotData.Skill3 != 0 {
			avatarSkillDepotData.Skills = append(avatarSkillDepotData.Skills, avatarSkillDepotData.Skill3)
		}
		if avatarSkillDepotData.Skill4 != 0 {
			avatarSkillDepotData.Skills = append(avatarSkillDepotData.Skills, avatarSkillDepotData.Skill4)
		}
		avatarSkillDepotData.Talents = make([]int32, 0)
		if avatarSkillDepotData.Talent1 != 0 {
			avatarSkillDepotData.Talents = append(avatarSkillDepotData.Talents, avatarSkillDepotData.Talent1)
		}
		if avatarSkillDepotData.Talent2 != 0 {
			avatarSkillDepotData.Talents = append(avatarSkillDepotData.Talents, avatarSkillDepotData.Talent2)
		}
		if avatarSkillDepotData.Talent3 != 0 {
			avatarSkillDepotData.Talents = append(avatarSkillDepotData.Talents, avatarSkillDepotData.Talent3)
		}
		if avatarSkillDepotData.Talent4 != 0 {
			avatarSkillDepotData.Talents = append(avatarSkillDepotData.Talents, avatarSkillDepotData.Talent4)
		}
		if avatarSkillDepotData.Talent5 != 0 {
			avatarSkillDepotData.Talents = append(avatarSkillDepotData.Talents, avatarSkillDepotData.Talent5)
		}
		if avatarSkillDepotData.Talent6 != 0 {
			avatarSkillDepotData.Talents = append(avatarSkillDepotData.Talents, avatarSkillDepotData.Talent6)
		}
		avatarSkillDepotData.InherentProudSkillOpens = make([]*InherentProudSkillOpens, 0)
		if avatarSkillDepotData.ProudSkill1GroupId != 0 {
			avatarSkillDepotData.InherentProudSkillOpens = append(avatarSkillDepotData.InherentProudSkillOpens, &InherentProudSkillOpens{
				ProudSkillGroupId:      avatarSkillDepotData.ProudSkill1GroupId,
				NeedAvatarPromoteLevel: avatarSkillDepotData.ProudSkill1NeedAvatarPromoteLevel,
			})
		}
		if avatarSkillDepotData.ProudSkill2GroupId != 0 {
			avatarSkillDepotData.InherentProudSkillOpens = append(avatarSkillDepotData.InherentProudSkillOpens, &InherentProudSkillOpens{
				ProudSkillGroupId:      avatarSkillDepotData.ProudSkill2GroupId,
				NeedAvatarPromoteLevel: avatarSkillDepotData.ProudSkill2NeedAvatarPromoteLevel,
			})
		}
		if avatarSkillDepotData.ProudSkill3GroupId != 0 {
			avatarSkillDepotData.InherentProudSkillOpens = append(avatarSkillDepotData.InherentProudSkillOpens, &InherentProudSkillOpens{
				ProudSkillGroupId:      avatarSkillDepotData.ProudSkill3GroupId,
				NeedAvatarPromoteLevel: avatarSkillDepotData.ProudSkill3NeedAvatarPromoteLevel,
			})
		}
		if avatarSkillDepotData.ProudSkill4GroupId != 0 {
			avatarSkillDepotData.InherentProudSkillOpens = append(avatarSkillDepotData.InherentProudSkillOpens, &InherentProudSkillOpens{
				ProudSkillGroupId:      avatarSkillDepotData.ProudSkill4GroupId,
				NeedAvatarPromoteLevel: avatarSkillDepotData.ProudSkill4NeedAvatarPromoteLevel,
			})
		}
		if avatarSkillDepotData.ProudSkill5GroupId != 0 {
			avatarSkillDepotData.InherentProudSkillOpens = append(avatarSkillDepotData.InherentProudSkillOpens, &InherentProudSkillOpens{
				ProudSkillGroupId:      avatarSkillDepotData.ProudSkill5GroupId,
				NeedAvatarPromoteLevel: avatarSkillDepotData.ProudSkill5NeedAvatarPromoteLevel,
			})
		}
		avatarSkillDepotData.AbilityHashCodeList = make([]int32, 0)
		if avatarSkillDepotData.SkillDepotAbilityGroup != "" {
			config := playerAbilities[avatarSkillDepotData.SkillDepotAbilityGroup]
			if config != nil {
				for _, targetAbility := range config.TargetAbilities {
					avatarSkillDepotData.AbilityHashCodeList = append(avatarSkillDepotData.AbilityHashCodeList, endec.Hk4eAbilityHashCode(targetAbility.AbilityName))
				}
			}
		}
		g.AvatarSkillDepotDataMap[avatarSkillDepotData.AvatarSkillDepotId] = avatarSkillDepotData
	}
	logger.Info("AvatarSkillDepotData count: %v", len(g.AvatarSkillDepotDataMap))
}

func GetAvatarSkillDepotDataById(avatarSkillDepotId int32) *AvatarSkillDepotData {
	return CONF.AvatarSkillDepotDataMap[avatarSkillDepotId]
}

func GetAvatarSkillDepotDataMap() map[int32]*AvatarSkillDepotData {
	return CONF.AvatarSkillDepotDataMap
}

func GetAvatarEnergySkillConfig(skillDepotId uint32) *AvatarSkillData {
	// 角色技能库配置
	avatarSkillDepotDataConfig, exist := CONF.AvatarSkillDepotDataMap[int32(skillDepotId)]
	if !exist {
		return nil
	}
	// 角色充能技配置
	avatarSkillDataConfig, exist := CONF.AvatarSkillDataMap[avatarSkillDepotDataConfig.EnergySkill]
	if !exist {
		return nil
	}
	return avatarSkillDataConfig
}

func GetAvatarInherentProudSkillList(skillDepotId uint32, promote uint8) []uint32 {
	// 角色技能库配置
	avatarSkillDepotDataConfig, exist := CONF.AvatarSkillDepotDataMap[int32(skillDepotId)]
	if !exist {
		return nil
	}
	inherentProudSkillList := make([]uint32, 0)
	for _, inherentProudSkillOpen := range avatarSkillDepotDataConfig.InherentProudSkillOpens {
		if inherentProudSkillOpen.NeedAvatarPromoteLevel >= int32(promote) {
			inherentProudSkill := uint32(0)
			inherentProudSkill = uint32(inherentProudSkillOpen.ProudSkillGroupId)*100 + 1
			inherentProudSkillList = append(inherentProudSkillList, inherentProudSkill)
		}
	}
	return inherentProudSkillList
}
