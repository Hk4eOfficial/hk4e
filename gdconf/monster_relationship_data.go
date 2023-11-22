package gdconf

import (
	"hk4e/pkg/logger"
)

// MonsterRelationshipData 怪物关联配置表
type MonsterRelationshipData struct {
	MonsterRelationshipId int32  `csv:"编号"`
	DropModel             string `csv:"掉落模型,omitempty"`
	MonsterModel          string `csv:"怪物模型,omitempty"`
}

func (g *GameDataConfig) loadMonsterRelationshipData() {
	g.MonsterRelationshipDataMap = make(map[int32]*MonsterRelationshipData)
	monsterRelationshipDataList := make([]*MonsterRelationshipData, 0)
	readTable[MonsterRelationshipData](g.txtPrefix+"MonsterRelationshipData.txt", &monsterRelationshipDataList)
	for _, monsterRelationshipData := range monsterRelationshipDataList {
		g.MonsterRelationshipDataMap[monsterRelationshipData.MonsterRelationshipId] = monsterRelationshipData
	}
	logger.Info("MonsterRelationshipData count: %v", len(g.MonsterRelationshipDataMap))
}

func GetMonsterRelationshipDataById(monsterRelationshipId int32) *MonsterRelationshipData {
	return CONF.MonsterRelationshipDataMap[monsterRelationshipId]
}

func GetMonsterRelationshipDataMap() map[int32]*MonsterRelationshipData {
	return CONF.MonsterRelationshipDataMap
}

func GetDropModelByMonsterModel(monsterModel string) string {
	for _, monsterRelationshipData := range CONF.MonsterRelationshipDataMap {
		if monsterRelationshipData.MonsterModel == monsterModel {
			return monsterRelationshipData.DropModel
		}
	}
	return ""
}
