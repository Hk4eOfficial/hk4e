package gdconf

import (
	"hk4e/pkg/logger"
)

type CostItem struct {
	ItemId    int32
	ItemCount int32
}

// ProudSkillData 天赋配置表
type ProudSkillData struct {
	ProudSkillId      int32 `csv:"技能ID"`
	ProudSkillGroupId int32 `csv:"技能组ID,omitempty"`
	Level             int32 `csv:"等级,omitempty"`
	Type              int32 `csv:"类型,omitempty"`
	CostSCoin         int32 `csv:"消耗金币,omitempty"`
	CostItem1Id       int32 `csv:"消耗道具1ID,omitempty"`
	CostItem1Count    int32 `csv:"消耗道具1数量,omitempty"`
	CostItem2Id       int32 `csv:"消耗道具2ID,omitempty"`
	CostItem2Count    int32 `csv:"消耗道具2数量,omitempty"`
	CostItem3Id       int32 `csv:"消耗道具3ID,omitempty"`
	CostItem3Count    int32 `csv:"消耗道具3数量,omitempty"`
	CostItem4Id       int32 `csv:"消耗道具4ID,omitempty"`
	CostItem4Count    int32 `csv:"消耗道具4数量,omitempty"`

	CostItemList []*CostItem
}

func (g *GameDataConfig) loadProudSkillData() {
	g.ProudSkillDataMap = make(map[int32]map[int32]*ProudSkillData)
	proudSkillDataList := make([]*ProudSkillData, 0)
	readTable[ProudSkillData](g.txtPrefix+"ProudSkillData.txt", &proudSkillDataList)
	for _, proudSkillData := range proudSkillDataList {
		proudSkillData.CostItemList = make([]*CostItem, 0)
		if proudSkillData.CostItem1Id != 0 {
			proudSkillData.CostItemList = append(proudSkillData.CostItemList, &CostItem{
				ItemId:    proudSkillData.CostItem1Id,
				ItemCount: proudSkillData.CostItem1Count,
			})
		}
		if proudSkillData.CostItem2Id != 0 {
			proudSkillData.CostItemList = append(proudSkillData.CostItemList, &CostItem{
				ItemId:    proudSkillData.CostItem2Id,
				ItemCount: proudSkillData.CostItem2Count,
			})
		}
		if proudSkillData.CostItem3Id != 0 {
			proudSkillData.CostItemList = append(proudSkillData.CostItemList, &CostItem{
				ItemId:    proudSkillData.CostItem3Id,
				ItemCount: proudSkillData.CostItem3Count,
			})
		}
		if proudSkillData.CostItem4Id != 0 {
			proudSkillData.CostItemList = append(proudSkillData.CostItemList, &CostItem{
				ItemId:    proudSkillData.CostItem4Id,
				ItemCount: proudSkillData.CostItem4Count,
			})
		}
		_, exist := g.ProudSkillDataMap[proudSkillData.ProudSkillGroupId]
		if !exist {
			g.ProudSkillDataMap[proudSkillData.ProudSkillGroupId] = make(map[int32]*ProudSkillData)
		}
		g.ProudSkillDataMap[proudSkillData.ProudSkillGroupId][proudSkillData.Level] = proudSkillData
	}
	logger.Info("ProudSkillData count: %v", len(g.ProudSkillDataMap))
}

func GetProudSkillDataByGroupIdAndLevel(groupId int32, level int32) *ProudSkillData {
	levelMap, exist := CONF.ProudSkillDataMap[groupId]
	if !exist {
		return nil
	}
	return levelMap[level]
}
