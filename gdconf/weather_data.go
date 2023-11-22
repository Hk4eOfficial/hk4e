package gdconf

import (
	"hk4e/pkg/logger"
)

// WeatherData 天气配置表
type WeatherData struct {
	WeatherId         int32  `csv:"ID"`
	WeatherAreaId     int32  `csv:"JSON天气区域ID,omitempty"`
	GadgetID          int32  `csv:"GadgetID,omitempty"`
	DefaultOpen       int32  `csv:"默认是否开启,omitempty"`
	TemplateName      string `csv:"TemplateName,omitempty"`
	Priority          int32  `csv:"Priority,omitempty"`
	DefaultWeather    int32  `csv:"DefaultWeather,omitempty"`
	UseDefaultWeather int32  `csv:"是否固定使用DefaultWeather,omitempty"`
	SceneId           int32  `csv:"场景ID,omitempty"`
}

func (g *GameDataConfig) loadWeatherData() {
	g.WeatherDataMap = make(map[int32]*WeatherData)
	weatherDataList := make([]*WeatherData, 0)
	readTable[WeatherData](g.txtPrefix+"WeatherData.txt", &weatherDataList)
	for _, weatherData := range weatherDataList {
		g.WeatherDataMap[weatherData.WeatherId] = weatherData
	}
	logger.Info("WeatherData count: %v", len(g.WeatherDataMap))
}

func GetWeatherDataById(weatherId int32) *WeatherData {
	return CONF.WeatherDataMap[weatherId]
}

func GetWeatherDataMap() map[int32]*WeatherData {
	return CONF.WeatherDataMap
}
