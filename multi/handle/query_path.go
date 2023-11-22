package handle

import (
	"bytes"
	"encoding/gob"
	"os"

	"hk4e/pkg/alg"
	"hk4e/pkg/logger"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"

	pb "google.golang.org/protobuf/proto"
)

func (h *Handle) ConvPbVecToMeshVec(pbVec *proto.Vector) alg.MeshVector {
	return alg.MeshVector{
		X: int16(pbVec.X),
		Y: int16(pbVec.Y),
		Z: int16(pbVec.Z),
	}
}

func (h *Handle) ConvMeshVecToPbVec(meshVec alg.MeshVector) *proto.Vector {
	return &proto.Vector{
		X: float32(meshVec.X),
		Y: float32(meshVec.Y),
		Z: float32(meshVec.Z),
	}
}

func (h *Handle) ConvPbVecListToMeshVecList(pbVecList []*proto.Vector) []alg.MeshVector {
	ret := make([]alg.MeshVector, 0)
	for _, pbVec := range pbVecList {
		ret = append(ret, h.ConvPbVecToMeshVec(pbVec))
	}
	return ret
}

func (h *Handle) ConvMeshVecListToPbVecList(meshVecList []alg.MeshVector) []*proto.Vector {
	ret := make([]*proto.Vector, 0)
	for _, meshVec := range meshVecList {
		ret = append(ret, h.ConvMeshVecToPbVec(meshVec))
	}
	return ret
}

func (h *Handle) QueryPath(userId uint32, gateAppId string, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.QueryPathReq)
	logger.Debug("query path req: %v, uid: %v, gateAppId: %v", req, userId, gateAppId)
	var ok = false
	var path []alg.MeshVector = nil
	for _, destinationPos := range req.DestinationPos {
		ok, path = h.worldStatic.Pathfinding(h.ConvPbVecToMeshVec(req.SourcePos), h.ConvPbVecToMeshVec(destinationPos))
		if ok {
			break
		}
	}
	if !ok {
		queryPathRsp := &proto.QueryPathRsp{
			QueryId:     req.QueryId,
			QueryStatus: proto.QueryPathRsp_STATUS_SUCC,
			Corners:     []*proto.Vector{req.DestinationPos[0]},
		}
		h.SendMsg(cmd.QueryPathRsp, userId, gateAppId, queryPathRsp)
		return
	}
	queryPathRsp := &proto.QueryPathRsp{
		QueryId:     req.QueryId,
		QueryStatus: proto.QueryPathRsp_STATUS_SUCC,
		Corners:     h.ConvMeshVecListToPbVecList(path),
	}
	h.SendMsg(cmd.QueryPathRsp, userId, gateAppId, queryPathRsp)
}

func (h *Handle) ObstacleModifyNotify(userId uint32, gateAppId string, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.ObstacleModifyNotify)
	logger.Debug("obstacle modify req: %v, uid: %v, gateAppId: %v", req, userId, gateAppId)
}

type WorldStatic struct {
	// x y z -> if terrain exist
	terrain map[alg.MeshVector]bool
}

func NewWorldStatic() (r *WorldStatic) {
	r = new(WorldStatic)
	r.terrain = make(map[alg.MeshVector]bool)
	return r
}

func (w *WorldStatic) InitTerrain() bool {
	data, err := os.ReadFile("./world_terrain.bin")
	if err != nil {
		logger.Error("read world terrain file error: %v", err)
		return false
	}
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err = decoder.Decode(&w.terrain)
	if err != nil {
		logger.Error("unmarshal world terrain data error: %v", err)
		return false
	}
	return true
}

func (w *WorldStatic) SaveTerrain() bool {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(w.terrain)
	if err != nil {
		logger.Error("marshal world terrain data error: %v", err)
		return false
	}
	err = os.WriteFile("./world_terrain.bin", buffer.Bytes(), 0644)
	if err != nil {
		logger.Error("write world terrain file error: %v", err)
		return false
	}
	return true
}

func (w *WorldStatic) GetTerrain(x int16, y int16, z int16) (exist bool) {
	pos := alg.MeshVector{
		X: x,
		Y: y,
		Z: z,
	}
	exist = w.terrain[pos]
	return exist
}

func (w *WorldStatic) SetTerrain(x int16, y int16, z int16) {
	pos := alg.MeshVector{
		X: x,
		Y: y,
		Z: z,
	}
	w.terrain[pos] = true
}

func (w *WorldStatic) Pathfinding(startPos alg.MeshVector, endPos alg.MeshVector) (bool, []alg.MeshVector) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error("pathfinding error, panic, startPos: %v, endPos: %v", startPos, endPos)
		}
	}()
	bfs := alg.NewBFS()
	bfs.InitMap(
		w.terrain,
		startPos,
		endPos,
		0,
	)
	pathVectorList := bfs.Pathfinding()
	if pathVectorList == nil {
		logger.Error("could not find path")
		return false, nil
	}
	logger.Debug("find path success, path: %v", pathVectorList)
	return true, pathVectorList
}
