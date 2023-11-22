package gdconf

import (
	"hk4e/pkg/logger"
)

// WeatherTemplate 天气模版配置表
type WeatherTemplate struct {
	TemplateName string `csv:"天气模板名"`
	Weather      int32  `csv:"天气,omitempty"`
	Sunny        int32  `csv:"晴,omitempty"`
	Cloudy       int32  `csv:"多云,omitempty"`
	Rain         int32  `csv:"雨,omitempty"`
	ThunderStorm int32  `csv:"雷雨,omitempty"`
	Snow         int32  `csv:"雪,omitempty"`
	Mist         int32  `csv:"雾,omitempty"`
	Desert       int32  `csv:"沙漠,omitempty"`
}

func (g *GameDataConfig) loadWeatherTemplateData() {
	g.WeatherTemplateMap = make(map[string]map[int32]*WeatherTemplate)
	weatherTemplateList := make([]*WeatherTemplate, 0)
	readTable[WeatherTemplate](g.txtPrefix+"WeatherTemplate.txt", &weatherTemplateList)
	for _, weatherTemplate := range weatherTemplateList {
		_, exist := g.WeatherTemplateMap[weatherTemplate.TemplateName]
		if !exist {
			g.WeatherTemplateMap[weatherTemplate.TemplateName] = make(map[int32]*WeatherTemplate)
		}
		g.WeatherTemplateMap[weatherTemplate.TemplateName][weatherTemplate.Weather] = weatherTemplate
	}
	logger.Info("WeatherTemplate count: %v", len(g.WeatherTemplateMap))
}

func GetWeatherTemplateByTemplateNameAndWeather(templateName string, weather int32) *WeatherTemplate {
	value, exist := CONF.WeatherTemplateMap[templateName]
	if !exist {
		return nil
	}
	return value[weather]
}

func GetWeatherTemplateMap() map[string]map[int32]*WeatherTemplate {
	return CONF.WeatherTemplateMap
}
