package game

import (
	"time"

	"hk4e/gs/model"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"

	pb "google.golang.org/protobuf/proto"
)

/************************************************** 接口请求 **************************************************/

func (g *Game) GetShopmallDataReq(player *model.Player, payloadMsg pb.Message) {
	getShopmallDataRsp := &proto.GetShopmallDataRsp{
		ShopTypeList: []uint32{900, 1052, 902, 1001, 903},
	}
	g.SendMsg(cmd.GetShopmallDataRsp, player.PlayerId, player.ClientSeq, getShopmallDataRsp)
}

func (g *Game) GetShopReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.GetShopReq)
	shopType := req.ShopType

	if shopType != 1001 {
		return
	}

	nextRefreshTime := uint32(time.Now().Add(time.Hour * 24 * 30).Unix())

	getShopRsp := &proto.GetShopRsp{
		Shop: &proto.Shop{
			GoodsList: []*proto.ShopGoods{
				{
					MinLevel:        1,
					EndTime:         2051193600,
					Hcoin:           160,
					GoodsId:         102001,
					NextRefreshTime: nextRefreshTime,
					MaxLevel:        99,
					BeginTime:       1575129600,
					GoodsItem: &proto.ItemParam{
						ItemId: 223,
						Count:  1,
					},
				},
				{
					MinLevel:        1,
					EndTime:         2051193600,
					Hcoin:           160,
					GoodsId:         102002,
					NextRefreshTime: nextRefreshTime,
					MaxLevel:        99,
					BeginTime:       1575129600,
					GoodsItem: &proto.ItemParam{
						ItemId: 224,
						Count:  1,
					},
				},
			},
			NextRefreshTime: nextRefreshTime,
			ShopType:        1001,
		},
	}
	g.SendMsg(cmd.GetShopRsp, player.PlayerId, player.ClientSeq, getShopRsp)
}

func (g *Game) BuyGoodsReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.BuyGoodsReq)
	buyItemId := req.Goods.GoodsItem.ItemId
	buyItemCount := req.BuyCount
	costHcoinCount := req.Goods.Hcoin * buyItemCount

	if buyItemId != 223 && buyItemId != 224 {
		return
	}

	if g.GetPlayerItemCount(player.PlayerId, 201) < costHcoinCount {
		return
	}
	ok := g.CostPlayerItem(player.PlayerId, []*ChangeItem{{ItemId: 201, ChangeCount: costHcoinCount}})
	if !ok {
		return
	}

	g.AddPlayerItem(player.PlayerId, []*ChangeItem{{ItemId: buyItemId, ChangeCount: buyItemCount}}, proto.ActionReasonType_ACTION_REASON_SHOP)
	req.Goods.BoughtNum = g.GetPlayerItemCount(player.PlayerId, buyItemId)

	buyGoodsRsp := &proto.BuyGoodsRsp{
		ShopType:  req.ShopType,
		BuyCount:  req.BuyCount,
		GoodsList: []*proto.ShopGoods{req.Goods},
	}
	g.SendMsg(cmd.BuyGoodsRsp, player.PlayerId, player.ClientSeq, buyGoodsRsp)
}

func (g *Game) McoinExchangeHcoinReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.McoinExchangeHcoinReq)
	if req.Hcoin != req.McoinCost {
		return
	}
	count := req.Hcoin

	if g.GetPlayerItemCount(player.PlayerId, 203) < count {
		return
	}
	ok := g.CostPlayerItem(player.PlayerId, []*ChangeItem{{ItemId: 203, ChangeCount: count}})
	if !ok {
		return
	}

	g.AddPlayerItem(player.PlayerId, []*ChangeItem{{ItemId: 201, ChangeCount: count}}, proto.ActionReasonType_ACTION_REASON_SHOP)

	mcoinExchangeHcoinRsp := &proto.McoinExchangeHcoinRsp{
		Hcoin:     req.Hcoin,
		McoinCost: req.McoinCost,
	}
	g.SendMsg(cmd.McoinExchangeHcoinRsp, player.PlayerId, player.ClientSeq, mcoinExchangeHcoinRsp)
}

/************************************************** 游戏功能 **************************************************/

/************************************************** 打包封装 **************************************************/
