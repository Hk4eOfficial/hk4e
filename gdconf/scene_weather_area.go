package gdconf

import (
	"os"
	"strconv"

	"hk4e/pkg/logger"

	"github.com/hjson/hjson-go/v4"
)

// SceneWeatherArea 场景天气区域配置表
type SceneWeatherArea struct {
	AreaId int32        `json:"area_id"` // 天气区域id
	Points []*AreaPoint `json:"points"`  // 多边形平面顶点数组
}

type AreaPoint struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

func (g *GameDataConfig) loadSceneWeatherArea() {
	g.SceneWeatherAreaMap = make(map[int32]map[int32]*SceneWeatherArea)
	sceneLuaPrefix := g.luaPrefix + "scene/"
	count := 0
	for _, sceneData := range g.SceneDataMap {
		sceneId := sceneData.SceneId
		sceneIdStr := strconv.Itoa(int(sceneId))
		// 读取场景天气区域
		fileData, err := os.ReadFile(sceneLuaPrefix + sceneIdStr + "/scene" + sceneIdStr + "_weather_areas.json")
		if err != nil {
			// 有些场景没有天气区域是正常情况
			// logger.Error("open file error: %v, sceneId: %v", err, sceneId)
			continue
		}
		sceneWeatherAreaList := make([]*SceneWeatherArea, 0)
		err = hjson.Unmarshal(fileData, &sceneWeatherAreaList)
		if err != nil {
			logger.Error("parse file error: %v, sceneId: %v", err, sceneId)
			continue
		}
		// 记录每个天气区域
		for _, area := range sceneWeatherAreaList {
			_, exist := g.SceneWeatherAreaMap[sceneId]
			if !exist {
				g.SceneWeatherAreaMap[sceneId] = make(map[int32]*SceneWeatherArea, len(area.Points))
			}
			g.SceneWeatherAreaMap[sceneId][area.AreaId] = area
			count++
		}
	}
	logger.Info("SceneWeatherArea count: %v", count)
}

func GetSceneWeatherAreaMapBySceneIdAndWeatherAreaId(sceneId int32, weatherAreaId int32) *SceneWeatherArea {
	value, exist := CONF.SceneWeatherAreaMap[sceneId]
	if !exist {
		return nil
	}
	return value[weatherAreaId]
}

func GetSceneWeatherAreaMap() map[int32]map[int32]*SceneWeatherArea {
	return CONF.SceneWeatherAreaMap
}
