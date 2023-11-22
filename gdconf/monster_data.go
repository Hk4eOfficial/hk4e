package gdconf

import (
	"hk4e/pkg/logger"
)

type HpDrop struct {
	Id        int32 // ID
	HpPercent int32 // 血量百分比
}

// MonsterData 怪物配置表
type MonsterData struct {
	MonsterId      int32   `csv:"ID"`
	Name           string  `csv:"名称$text_name_Name,omitempty"`
	HpBase         float32 `csv:"基础生命值,omitempty"`
	AttackBase     float32 `csv:"基础攻击力,omitempty"`
	DefenseBase    float32 `csv:"基础防御力,omitempty"`
	Critical       float32 `csv:"暴击率,omitempty"`
	CriticalHurt   float32 `csv:"暴击伤害,omitempty"`
	Drop1Id        int32   `csv:"[掉落]1ID,omitempty"`
	Drop1HpPercent int32   `csv:"[掉落]1血量百分比,omitempty"`
	Drop2Id        int32   `csv:"[掉落]2ID,omitempty"`
	Drop2HpPercent int32   `csv:"[掉落]2血量百分比,omitempty"`
	Drop3Id        int32   `csv:"[掉落]3ID,omitempty"`
	Drop3HpPercent int32   `csv:"[掉落]3血量百分比,omitempty"`
	KillDropId     int32   `csv:"击杀掉落ID,omitempty"`

	HpDropList []*HpDrop // 血量掉落列表
}

func (g *GameDataConfig) loadMonsterData() {
	g.MonsterDataMap = make(map[int32]*MonsterData)
	monsterDataList := make([]*MonsterData, 0)
	readTable[MonsterData](g.txtPrefix+"MonsterData.txt", &monsterDataList)
	for _, monsterData := range monsterDataList {
		monsterData.HpDropList = make([]*HpDrop, 0)
		if monsterData.Drop1Id != 0 {
			monsterData.HpDropList = append(monsterData.HpDropList, &HpDrop{
				Id:        monsterData.Drop1Id,
				HpPercent: monsterData.Drop1HpPercent,
			})
		}
		if monsterData.Drop2Id != 0 {
			monsterData.HpDropList = append(monsterData.HpDropList, &HpDrop{
				Id:        monsterData.Drop2Id,
				HpPercent: monsterData.Drop2HpPercent,
			})
		}
		if monsterData.Drop3Id != 0 {
			monsterData.HpDropList = append(monsterData.HpDropList, &HpDrop{
				Id:        monsterData.Drop3Id,
				HpPercent: monsterData.Drop3HpPercent,
			})
		}
		g.MonsterDataMap[monsterData.MonsterId] = monsterData
	}
	logger.Info("MonsterData count: %v", len(g.MonsterDataMap))
}

func GetMonsterDataById(monsterId int32) *MonsterData {
	return CONF.MonsterDataMap[monsterId]
}

func GetMonsterDataMap() map[int32]*MonsterData {
	return CONF.MonsterDataMap
}

// TODO 成长属性要读表

func (m *MonsterData) GetBaseHpByLevel(level uint8) float32 {
	return m.HpBase * float32(level)
}

func (m *MonsterData) GetBaseAttackByLevel(level uint8) float32 {
	return m.AttackBase * float32(level)
}

func (m *MonsterData) GetBaseDefenseByLevel(level uint8) float32 {
	return m.DefenseBase * float32(level)
}
