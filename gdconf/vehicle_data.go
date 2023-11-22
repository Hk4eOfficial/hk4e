package gdconf

import (
	"fmt"
	"os"
	"strings"

	"hk4e/pkg/logger"

	"github.com/hjson/hjson-go/v4"
)

// VehicleData 载具配置表
type VehicleData struct {
	VehicleId int32  `csv:"ID"`
	TextName  string `csv:"名称$text_name_Name,omitempty"`

	ConfigGadgetVehicle *ConfigGadgetVehicle
}

type ConfigGadgetVehicle struct {
	Vehicle *ConfigVehicle       `json:"vehicle"` // 载具
	Combat  *ConfigVehicleCombat `json:"combat"`  // 战斗
}

type ConfigVehicle struct {
	VehicleType  string                `json:"vehicleType"`  // 载具类型
	DefaultLevel int32                 `json:"defaultLevel"` // 默认等级
	MaxSeatCount int32                 `json:"maxSeatCount"` // 最大座位数
	Stamina      *ConfigVehicleStamina `json:"stamina"`      // 耐力
}

type ConfigVehicleStamina struct {
	StaminaUpperLimit      float32 `json:"staminaUpperLimit"`      // 耐力上限
	StaminaRecoverSpeed    float32 `json:"staminaRecoverSpeed"`    // 耐力回复速度
	StaminaRecoverWaitTime float32 `json:"staminaRecoverWaitTime"` // 耐力回复等待时间
	ExtraStaminaUpperLimit float32 `json:"extraStaminaUpperLimit"` // 额外耐力上限
	SprintStaminaCost      float32 `json:"sprintStaminaCost"`      // 冲刺时耐力消耗
	DashStaminaCost        float32 `json:"dashStaminaCost"`        // 猛冲时耐力消耗
}

type ConfigVehicleCombat struct {
	Property *ConfigVehicleCombatProperty // 属性
}

type ConfigVehicleCombatProperty struct {
	HP          float32 `json:"HP"`          // 血量
	Attack      float32 `json:"attack"`      // 攻击力
	DefenseBase float32 `json:"defenseBase"` // 防御力
	Weight      float32 `json:"weight"`      // 重量
}

func (g *GameDataConfig) loadVehicleData() {
	g.VehicleDataMap = make(map[int32]*VehicleData)
	configGadgetVehiclesMap := make(map[string]map[string]*ConfigGadgetVehicle)
	vehicleDataList := make([]*VehicleData, 0)
	readTable[VehicleData](g.txtPrefix+"GadgetData_Vehicle.txt", &vehicleDataList)
	for _, vehicleData := range vehicleDataList {
		vehicleNameSplit := strings.Split(vehicleData.TextName, "_")
		if len(vehicleNameSplit) < 3 {
			info := fmt.Sprintf("vehicle text name split len less 3, TextName: %v", vehicleData.TextName)
			panic(info)
		}
		// 载具类型
		vehicleType := vehicleNameSplit[0]
		// 先判断一下是否已读取过同类型的载具配置
		configGadgetVehicles, ok := configGadgetVehiclesMap[vehicleType]
		if !ok {
			// 读取载具config
			fileData, err := os.ReadFile(g.jsonPrefix + "gadget/ConfigGadget_Vehicle_" + vehicleType + ".json")
			if err != nil {
				info := fmt.Sprintf("open file error: %v, vehicleType: %v", err, vehicleType)
				panic(info)
			}
			err = hjson.Unmarshal(fileData, &configGadgetVehicles)
			if err != nil {
				info := fmt.Sprintf("parse file error: %v, vehicleType: %v", err, vehicleType)
				panic(info)
			}
			configGadgetVehiclesMap[vehicleType] = configGadgetVehicles
		}
		// 载具config配置是否存在
		configVehicle, ok := configGadgetVehicles[vehicleData.TextName]
		if !ok {
			logger.Info("can not find any config of vehicle, vehicleId: %v", vehicleData.VehicleId)
		}
		vehicleData.ConfigGadgetVehicle = configVehicle

		g.VehicleDataMap[vehicleData.VehicleId] = vehicleData
	}
	logger.Info("VehicleData count: %v", len(g.VehicleDataMap))
}

func GetVehicleDataById(vehicleId int32) *VehicleData {
	return CONF.VehicleDataMap[vehicleId]
}

func GetVehicleDataMap() map[int32]*VehicleData {
	return CONF.VehicleDataMap
}
