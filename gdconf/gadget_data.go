package gdconf

import (
	"hk4e/pkg/logger"
)

// GadgetData 物件配置表
type GadgetData struct {
	GadgetId        int32  `csv:"ID"`
	Name            string `csv:"名称$text_name_Name,omitempty"`
	PrefabPath      string `csv:"Prefab路径,omitempty"`
	DefaultCamp     int32  `csv:"默认阵营,omitempty"`
	Type            int32  `csv:"类型,omitempty"`
	CanInteract     int32  `csv:"能否交互,omitempty"`
	VisionLevel     int32  `csv:"视距等级,omitempty"`
	ServerLuaScript string `csv:"服务器脚本,omitempty"`
}

func (g *GameDataConfig) loadGadgetData() {
	g.GadgetDataMap = make(map[int32]*GadgetData)
	fileNameList := []string{
		"GadgetData_AbilitySpecial.txt",
		"GadgetData_Affix.txt",
		"GadgetData_Avatar.txt",
		"GadgetData_Equip.txt",
		"GadgetData_FishingRod.txt",
		"GadgetData_Homeworld.txt",
		"GadgetData_Level.txt",
		"GadgetData_Monster.txt",
		"GadgetData_Quest.txt",
		"GadgetData_Vehicle.txt",
	}
	for _, fileName := range fileNameList {
		gadgetDataList := make([]*GadgetData, 0)
		readTable[GadgetData](g.txtPrefix+fileName, &gadgetDataList)
		for _, gadgetData := range gadgetDataList {
			g.GadgetDataMap[gadgetData.GadgetId] = gadgetData
		}
	}
	logger.Info("GadgetData count: %v", len(g.GadgetDataMap))
}

func GetGadgetDataById(gadgetId int32) *GadgetData {
	return CONF.GadgetDataMap[gadgetId]
}

func GetGadgetDataMap() map[int32]*GadgetData {
	return CONF.GadgetDataMap
}
