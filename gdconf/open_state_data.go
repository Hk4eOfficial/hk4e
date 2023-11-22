package gdconf

import (
	"hk4e/pkg/logger"
)

type OpenStateCond struct {
	Type  int32
	Param []int32
}

// OpenStateData 开放状态配置表
type OpenStateData struct {
	OpenStateId     int32 `csv:"ID"`
	DefaultOpen     int32 `csv:"默认是否开启,omitempty"`
	AllowClientReq  int32 `csv:"客户端能否发起功能开启,omitempty"`
	CondType1       int32 `csv:"条件1类型,omitempty"`
	CondType1Param  int32 `csv:"条件1参数,omitempty"`
	CondType1Param2 int32 `csv:"条件1参数2,omitempty"`
	CondType2       int32 `csv:"条件2类型,omitempty"`
	CondType2Param  int32 `csv:"条件2参数,omitempty"`
	CondType2Param2 int32 `csv:"条件2参数2,omitempty"`

	OpenStateCondList []*OpenStateCond
}

func (g *GameDataConfig) loadOpenStateData() {
	g.OpenStateDataMap = make(map[int32]*OpenStateData)
	openStateDataList := make([]*OpenStateData, 0)
	readTable[OpenStateData](g.txtPrefix+"OpenStateData.txt", &openStateDataList)
	for _, openStateData := range openStateDataList {
		openStateData.OpenStateCondList = make([]*OpenStateCond, 0)
		if openStateData.CondType1 != 0 {
			paramList := make([]int32, 0)
			if openStateData.CondType1Param != 0 {
				paramList = append(paramList, openStateData.CondType1Param)
			}
			if openStateData.CondType1Param2 != 0 {
				paramList = append(paramList, openStateData.CondType1Param2)
			}
			openStateData.OpenStateCondList = append(openStateData.OpenStateCondList, &OpenStateCond{
				Type:  openStateData.CondType1,
				Param: paramList,
			})
		}
		if openStateData.CondType2 != 0 {
			paramList := make([]int32, 0)
			if openStateData.CondType2Param != 0 {
				paramList = append(paramList, openStateData.CondType2Param)
			}
			if openStateData.CondType2Param2 != 0 {
				paramList = append(paramList, openStateData.CondType2Param2)
			}
			openStateData.OpenStateCondList = append(openStateData.OpenStateCondList, &OpenStateCond{
				Type:  openStateData.CondType2,
				Param: paramList,
			})
		}
		g.OpenStateDataMap[openStateData.OpenStateId] = openStateData
	}
	logger.Info("OpenStateData count: %v", len(g.OpenStateDataMap))
}

func GetOpenStateDataById(openStateId int32) *OpenStateData {
	return CONF.OpenStateDataMap[openStateId]
}

func GetOpenStateDataMap() map[int32]*OpenStateData {
	return CONF.OpenStateDataMap
}
