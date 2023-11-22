package gdconf

import (
	"hk4e/pkg/logger"
)

// RewardData 奖励配置表
type RewardData struct {
	RewardId         int32 `csv:"RewardID"`
	RewardItem1Id    int32 `csv:"Reward道具1ID,omitempty"`
	RewardItem1Count int32 `csv:"Reward道具1数量,omitempty"`
	RewardItem2Id    int32 `csv:"Reward道具2ID,omitempty"`
	RewardItem2Count int32 `csv:"Reward道具2数量,omitempty"`
	RewardItem3Id    int32 `csv:"Reward道具3ID,omitempty"`
	RewardItem3Count int32 `csv:"Reward道具3数量,omitempty"`
	RewardItem4Id    int32 `csv:"Reward道具4ID,omitempty"`
	RewardItem4Count int32 `csv:"Reward道具4数量,omitempty"`
	RewardItem5Id    int32 `csv:"Reward道具5ID,omitempty"`
	RewardItem5Count int32 `csv:"Reward道具5数量,omitempty"`
	RewardItem6Id    int32 `csv:"Reward道具6ID,omitempty"`
	RewardItem6Count int32 `csv:"Reward道具6数量,omitempty"`
	RewardItem7Id    int32 `csv:"Reward道具7ID,omitempty"`
	RewardItem7Count int32 `csv:"Reward道具7数量,omitempty"`
	RewardItem8Id    int32 `csv:"Reward道具8ID,omitempty"`
	RewardItem8Count int32 `csv:"Reward道具8数量,omitempty"`
	RewardItem9Id    int32 `csv:"Reward道具9ID,omitempty"`
	RewardItem9Count int32 `csv:"Reward道具9数量,omitempty"`

	RewardItemMap map[uint32]uint32
}

func (g *GameDataConfig) loadRewardData() {
	g.RewardDataMap = make(map[int32]*RewardData)
	rewardDataList := make([]*RewardData, 0)
	readTable[RewardData](g.txtPrefix+"RewardData.txt", &rewardDataList)
	for _, rewardData := range rewardDataList {
		// 奖励物品整合
		rewardData.RewardItemMap = map[uint32]uint32{
			uint32(rewardData.RewardItem1Id): uint32(rewardData.RewardItem1Count),
			uint32(rewardData.RewardItem2Id): uint32(rewardData.RewardItem2Count),
			uint32(rewardData.RewardItem3Id): uint32(rewardData.RewardItem3Count),
			uint32(rewardData.RewardItem4Id): uint32(rewardData.RewardItem4Count),
			uint32(rewardData.RewardItem5Id): uint32(rewardData.RewardItem5Count),
			uint32(rewardData.RewardItem6Id): uint32(rewardData.RewardItem6Count),
			uint32(rewardData.RewardItem7Id): uint32(rewardData.RewardItem7Count),
			uint32(rewardData.RewardItem8Id): uint32(rewardData.RewardItem8Count),
			uint32(rewardData.RewardItem9Id): uint32(rewardData.RewardItem9Count),
		}
		for itemId, count := range rewardData.RewardItemMap {
			// 两个值都不能为0
			if itemId == 0 || count == 0 {
				delete(rewardData.RewardItemMap, itemId)
			}
		}
		g.RewardDataMap[rewardData.RewardId] = rewardData
	}
	logger.Info("RewardData count: %v", len(g.RewardDataMap))
}

func GetRewardDataById(rewardID int32) *RewardData {
	return CONF.RewardDataMap[rewardID]
}

func GetRewardDataMap() map[int32]*RewardData {
	return CONF.RewardDataMap
}
