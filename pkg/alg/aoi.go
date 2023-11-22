package alg

import (
	"math"
)

// AoiManager aoi管理模块
type AoiManager struct {
	// 区域边界坐标
	minX    int32
	maxX    int32
	minY    int32
	maxY    int32
	minZ    int32
	maxZ    int32
	numX    uint32           // x方向格子的数量
	numY    uint32           // y方向的格子数量
	numZ    uint32           // z方向的格子数量
	gridMap map[uint32]*Grid // 当前区域中都有哪些格子 key:gid value:格子对象
}

func NewAoiManager() (r *AoiManager) {
	r = new(AoiManager)
	r.gridMap = make(map[uint32]*Grid)
	return r
}

// SetAoiRange 设置aoi区域边界坐标
func (a *AoiManager) SetAoiRange(minX, maxX, minY, maxY, minZ, maxZ int32) bool {
	if minX >= maxX || minY >= maxY || minZ >= maxZ {
		return false
	}
	a.minX = minX
	a.maxX = maxX
	a.minY = minY
	a.maxY = maxY
	a.minZ = minZ
	a.maxZ = maxZ
	return true
}

// Init3DRectAoiManager 初始化3D矩形aoi区域
func (a *AoiManager) Init3DRectAoiManager(numX, numY, numZ uint32, delay bool) bool {
	if numX <= 0 || numY <= 0 || numZ <= 0 {
		return false
	}
	if uint64(numX)*uint64(numY)*uint64(numZ) >= math.MaxUint32 {
		return false
	}
	a.numX = numX
	a.numY = numY
	a.numZ = numZ
	if !delay {
		// 初始化aoi区域中所有的格子
		for x := uint32(0); x < a.numX; x++ {
			for y := uint32(0); y < a.numY; y++ {
				for z := uint32(0); z < a.numZ; z++ {
					// 利用格子坐标得到格子id gid从0开始按xzy的顺序增长
					gid := y*(a.numX*a.numZ) + z*a.numX + x
					// 初始化一个格子放在aoi中的map里 key是当前格子的id
					grid := NewGrid(gid)
					a.gridMap[gid] = grid
				}
			}
		}
	}
	return true
}

func (a *AoiManager) GetGrid(gid uint32) *Grid {
	grid, exist := a.gridMap[gid]
	if !exist {
		if gid >= a.numX*a.numY*a.numZ {
			return nil
		}
		grid = NewGrid(gid)
		a.gridMap[gid] = grid
	}
	return grid
}

// GridXLen 每个格子在x轴方向的长度
func (a *AoiManager) GridXLen() uint32 {
	return uint32(a.maxX-a.minX) / a.numX
}

// GridYLen 每个格子在y轴方向的长度
func (a *AoiManager) GridYLen() uint32 {
	return uint32(a.maxY-a.minY) / a.numY
}

// GridZLen 每个格子在z轴方向的长度
func (a *AoiManager) GridZLen() uint32 {
	return uint32(a.maxZ-a.minZ) / a.numZ
}

// GetGidByPos 通过坐标获取对应的格子id
func (a *AoiManager) GetGidByPos(x, y, z float32) uint32 {
	if !a.IsValidAoiPos(x, y, z) {
		return math.MaxUint32
	}
	gx := uint32(int32(x)-a.minX) / a.GridXLen()
	gy := uint32(int32(y)-a.minY) / a.GridYLen()
	gz := uint32(int32(z)-a.minZ) / a.GridZLen()
	return gy*(a.numX*a.numZ) + gz*a.numX + gx
}

// IsValidAoiPos 判断坐标是否存在于aoi区域内
func (a *AoiManager) IsValidAoiPos(x, y, z float32) bool {
	if (int32(x) > a.minX && int32(x) < a.maxX) &&
		(int32(y) > a.minY && int32(y) < a.maxY) &&
		(int32(z) > a.minZ && int32(z) < a.maxZ) {
		return true
	} else {
		return false
	}
}

// GetSurrGridListByGid 根据格子的gid得到当前周边的格子信息
func (a *AoiManager) GetSurrGridListByGid(gid uint32) []*Grid {
	gridList := make([]*Grid, 0, 27)
	// 判断grid是否存在
	grid := a.GetGrid(gid)
	if grid == nil {
		return nil
	}
	// 添加自己
	if grid != nil {
		gridList = append(gridList, grid)
	}
	// 根据gid得到当前格子所在的x轴编号
	idx := gid % (a.numX * a.numZ) % a.numX
	// 判断当前格子左边是否还有格子
	if idx > 0 {
		grid = a.GetGrid(gid - 1)
		if grid != nil {
			gridList = append(gridList, grid)
		}
	}
	// 判断当前格子右边是否还有格子
	if idx < a.numX-1 {
		grid = a.GetGrid(gid + 1)
		if grid != nil {
			gridList = append(gridList, grid)
		}
	}
	// 将x轴当前的格子都取出进行遍历 再分别得到每个格子的平面上下是否有格子
	// 得到当前x轴的格子id集合
	gidListX := make([]uint32, 0)
	for _, v := range gridList {
		gidListX = append(gidListX, v.gid)
	}
	// 遍历x轴格子
	for _, v := range gidListX {
		// 计算该格子的idz
		idz := v % (a.numX * a.numZ) / a.numX
		// 判断当前格子平面上方是否还有格子
		if idz > 0 {
			grid = a.GetGrid(v - a.numX)
			if grid != nil {
				gridList = append(gridList, grid)
			}
		}
		// 判断当前格子平面下方是否还有格子
		if idz < a.numZ-1 {
			grid = a.GetGrid(v + a.numX)
			if grid != nil {
				gridList = append(gridList, grid)
			}
		}
	}
	// 将xoz平面当前的格子都取出进行遍历 再分别得到每个格子的空间上下是否有格子
	// 得到当前xoz平面的格子id集合
	gidListXOZ := make([]uint32, 0)
	for _, v := range gridList {
		gidListXOZ = append(gidListXOZ, v.gid)
	}
	// 遍历xoz平面格子
	for _, v := range gidListXOZ {
		// 计算该格子的idy
		idy := v / (a.numX * a.numZ)
		// 判断当前格子空间上方是否还有格子
		if idy > 0 {
			grid = a.GetGrid(v - a.numX*a.numZ)
			if grid != nil {
				gridList = append(gridList, grid)
			}
		}
		// 判断当前格子空间下方是否还有格子
		if idy < a.numY-1 {
			grid = a.GetGrid(v + a.numX*a.numZ)
			if grid != nil {
				gridList = append(gridList, grid)
			}
		}
	}
	return gridList
}

// GetObjectListByPos 通过坐标得到周边格子内的全部object
func (a *AoiManager) GetObjectListByPos(x, y, z float32) map[int64]any {
	// 根据坐标得到当前坐标属于哪个格子id
	gid := a.GetGidByPos(x, y, z)
	if gid == math.MaxUint32 {
		return nil
	}
	// 根据格子id得到周边格子的信息
	gridList := a.GetSurrGridListByGid(gid)
	objectListLen := 0
	for _, v := range gridList {
		tmp := v.GetObjectList()
		objectListLen += len(tmp)
	}
	objectList := make(map[int64]any, objectListLen)
	for _, v := range gridList {
		tmp := v.GetObjectList()
		for kk, vv := range tmp {
			objectList[kk] = vv
		}
	}
	return objectList
}

// GetObjectListByGid 通过gid获取当前格子的全部object
func (a *AoiManager) GetObjectListByGid(gid uint32) map[int64]any {
	grid := a.GetGrid(gid)
	if grid == nil {
		return nil
	}
	objectList := grid.GetObjectList()
	return objectList
}

// AddObjectToGrid 添加一个object到一个格子中
func (a *AoiManager) AddObjectToGrid(objectId int64, object any, gid uint32) bool {
	grid := a.GetGrid(gid)
	if grid == nil {
		return false
	}
	grid.AddObject(objectId, object)
	return true
}

// RemoveObjectFromGrid 移除一个格子中的object
func (a *AoiManager) RemoveObjectFromGrid(objectId int64, gid uint32) bool {
	grid := a.GetGrid(gid)
	if grid == nil {
		return false
	}
	grid.RemoveObject(objectId)
	return true
}

// AddObjectToGridByPos 通过坐标添加一个object到一个格子中
func (a *AoiManager) AddObjectToGridByPos(objectId int64, object any, x, y, z float32) bool {
	gid := a.GetGidByPos(x, y, z)
	if gid == math.MaxUint32 {
		return false
	}
	return a.AddObjectToGrid(objectId, object, gid)
}

// RemoveObjectFromGridByPos 通过坐标把一个object从对应的格子中删除
func (a *AoiManager) RemoveObjectFromGridByPos(objectId int64, x, y, z float32) bool {
	gid := a.GetGidByPos(x, y, z)
	if gid == math.MaxUint32 {
		return false
	}
	return a.RemoveObjectFromGrid(objectId, gid)
}

func (a *AoiManager) Debug() map[uint32]*Grid {
	return a.gridMap
}

// Grid 地图格子
type Grid struct {
	gid       uint32        // 格子id
	objectMap map[int64]any // k:objectId v:对象
}

// NewGrid 初始化格子
func NewGrid(gid uint32) (r *Grid) {
	r = new(Grid)
	r.gid = gid
	r.objectMap = make(map[int64]any)
	return r
}

// GetGid 获取格子id
func (g *Grid) GetGid() uint32 {
	return g.gid
}

// AddObject 向格子中添加一个对象
func (g *Grid) AddObject(objectId int64, object any) {
	g.objectMap[objectId] = object
}

// RemoveObject 从格子中删除一个对象
func (g *Grid) RemoveObject(objectId int64) {
	delete(g.objectMap, objectId)
}

// GetObjectList 获取格子中所有对象
func (g *Grid) GetObjectList() map[int64]any {
	return g.objectMap
}
