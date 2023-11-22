package gdconf

import (
	"hk4e/pkg/logger"
)

// pubg世界物件

const (
	PubgWorldGadgetTypeIncAtk = 1 // 增加攻击力 参数1:攻击力
	PubgWorldGadgetTypeIncHp  = 2 // 恢复生命值 参数1:生命值
)

type PubgWorldGadgetData struct {
	WorldGadgetId int32    `csv:"WorldGadgetId"`
	GadgetId      int32    `csv:"GadgetId"`
	X             float32  `csv:"X"`
	Y             float32  `csv:"Y"`
	Z             float32  `csv:"Z"`
	Probability   int32    `csv:"Probability"`
	Type          int32    `csv:"Type"`
	Param         IntArray `csv:"Param"`
}

func (g *GameDataConfig) loadPubgWorldGadgetData() {
	g.PubgWorldGadgetDataMap = make(map[int32]*PubgWorldGadgetData)
	pubgWorldGadgetDataList := make([]*PubgWorldGadgetData, 0)
	readExtCsv[PubgWorldGadgetData](g.extPrefix+"PubgWorldGadgetData.csv", &pubgWorldGadgetDataList)
	for _, pubgWorldGadgetData := range pubgWorldGadgetDataList {
		g.PubgWorldGadgetDataMap[pubgWorldGadgetData.WorldGadgetId] = pubgWorldGadgetData
	}
	logger.Info("PubgWorldGadgetData count: %v", len(g.PubgWorldGadgetDataMap))
}

func GetPubgWorldGadgetDataById(worldGadgetId int32) *PubgWorldGadgetData {
	return CONF.PubgWorldGadgetDataMap[worldGadgetId]
}

func GetPubgWorldGadgetDataMap() map[int32]*PubgWorldGadgetData {
	return CONF.PubgWorldGadgetDataMap
}
