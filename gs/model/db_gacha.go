package model

type DbGacha struct {
	GachaPoolInfo map[uint32]*GachaPoolInfo
}

type GachaPoolInfo struct {
	GachaType       uint32 // 卡池类型
	OrangeTimes     uint32 // 5星保底计数
	PurpleTimes     uint32 // 4星保底计数
	MustGetUpOrange bool   // 是否5星大保底
	MustGetUpPurple bool   // 是否4星大保底
}

func (p *Player) GetDbGacha() *DbGacha {
	if p.DbGacha == nil {
		p.DbGacha = &DbGacha{
			GachaPoolInfo: map[uint32]*GachaPoolInfo{
				300: {
					// 温迪
					GachaType:       300,
					OrangeTimes:     0,
					PurpleTimes:     0,
					MustGetUpOrange: false,
					MustGetUpPurple: false,
				},
				400: {
					// 可莉
					GachaType:       400,
					OrangeTimes:     0,
					PurpleTimes:     0,
					MustGetUpOrange: false,
					MustGetUpPurple: false,
				},
				431: {
					// 阿莫斯之弓&天空之傲
					GachaType:       431,
					OrangeTimes:     0,
					PurpleTimes:     0,
					MustGetUpOrange: false,
					MustGetUpPurple: false,
				},
				201: {
					// 常驻
					GachaType:       201,
					OrangeTimes:     0,
					PurpleTimes:     0,
					MustGetUpOrange: false,
					MustGetUpPurple: false,
				},
			},
		}
	}
	return p.DbGacha
}
