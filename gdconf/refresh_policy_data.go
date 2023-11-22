package gdconf

import (
	"fmt"
	"strconv"
	"strings"

	"hk4e/pkg/logger"
)

const (
	RefreshTypeNone         = 0
	RefreshTypeAfterTime    = 1
	RefreshTypeDayTime      = 2
	RefreshTypeDayTimeRange = 3
	RefreshTypeDay          = 4
)

// RefreshPolicyData 刷新策略配置表
type RefreshPolicyData struct {
	RefreshId      int32  `csv:"刷新ID"`
	RefreshType    int32  `csv:"刷新方式,omitempty"`
	RefreshTimeStr string `csv:"刷新时间,omitempty"`

	RefreshTime      int32
	RefreshTimeRange [2]int32
}

func (g *GameDataConfig) loadRefreshPolicyData() {
	g.RefreshPolicyDataMap = make(map[int32]*RefreshPolicyData)
	refreshPolicyDataList := make([]*RefreshPolicyData, 0)
	readTable[RefreshPolicyData](g.txtPrefix+"RefreshPolicyData.txt", &refreshPolicyDataList)
	for _, refreshPolicyData := range refreshPolicyDataList {
		if refreshPolicyData.RefreshType < RefreshTypeNone || refreshPolicyData.RefreshType > RefreshTypeDay {
			info := fmt.Sprintf("invalid refresh type: %v", refreshPolicyData)
			panic(info)
		}
		if refreshPolicyData.RefreshType == RefreshTypeDayTimeRange {
			split := strings.Split(refreshPolicyData.RefreshTimeStr, ";")
			if len(split) != 2 {
				info := fmt.Sprintf("refresh time format error: %v", refreshPolicyData)
				panic(info)
			}
			startTime, err := strconv.Atoi(split[0])
			if err != nil {
				panic(err)
			}
			endTime, err := strconv.Atoi(split[1])
			if err != nil {
				panic(err)
			}
			refreshPolicyData.RefreshTimeRange = [2]int32{int32(startTime), int32(endTime)}
		} else if refreshPolicyData.RefreshType == RefreshTypeNone {
			refreshPolicyData.RefreshTime = 0
		} else {
			refreshTime, err := strconv.Atoi(refreshPolicyData.RefreshTimeStr)
			if err != nil {
				panic(err)
			}
			refreshPolicyData.RefreshTime = int32(refreshTime)
		}
		g.RefreshPolicyDataMap[refreshPolicyData.RefreshId] = refreshPolicyData
	}
	logger.Info("RefreshPolicyData count: %v", len(g.RefreshPolicyDataMap))
}

func GetRefreshPolicyDataById(refreshId int32) *RefreshPolicyData {
	return CONF.RefreshPolicyDataMap[refreshId]
}

func GetRefreshPolicyDataMap() map[int32]*RefreshPolicyData {
	return CONF.RefreshPolicyDataMap
}
