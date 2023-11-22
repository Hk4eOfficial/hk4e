package model

type WeatherInfo struct {
	WeatherAreaId uint32 // 天气区域id
	ClimateType   uint32 // 气候类型
}

func NewWeatherInfo() *WeatherInfo {
	return &WeatherInfo{
		WeatherAreaId: 0,
		ClimateType:   0,
	}
}
