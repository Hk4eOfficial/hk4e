package model

import (
	"time"
)

type DbSocial struct {
	Birthday        []uint8           // 生日
	NameCard        uint32            // 当前名片
	NameCardList    []uint32          // 已解锁名片列表
	FriendList      map[uint32]uint32 // 好友uid列表
	FriendApplyList map[uint32]uint32 // 好友申请uid列表
}

func (p *Player) GetDbSocial() *DbSocial {
	if p.DbSocial == nil {
		p.DbSocial = new(DbSocial)
	}
	if p.DbSocial.Birthday == nil {
		p.DbSocial.Birthday = []uint8{0, 0}
	}
	if p.DbSocial.NameCard == 0 {
		p.DbSocial.NameCard = 210001
	}
	if p.DbSocial.NameCardList == nil {
		p.DbSocial.NameCardList = []uint32{210001}
	}
	if p.DbSocial.FriendList == nil {
		p.DbSocial.FriendList = make(map[uint32]uint32)
	}
	if p.DbSocial.FriendApplyList == nil {
		p.DbSocial.FriendApplyList = make(map[uint32]uint32)
	}
	return p.DbSocial
}

func (s *DbSocial) GetBirthdayMonth() uint32 {
	return uint32(s.Birthday[0])
}

func (s *DbSocial) GetBirthdayDay() uint32 {
	return uint32(s.Birthday[1])
}

func (s *DbSocial) IsSetBirthday() bool {
	if s.Birthday[0] != 0 || s.Birthday[1] != 0 {
		return true
	}
	return false
}

func (s *DbSocial) SetBirthday(month uint32, day uint32) {
	s.Birthday[0] = uint8(month)
	s.Birthday[1] = uint8(day)
}

func (s *DbSocial) UnlockNameCard(nameCardId uint32) {
	for _, v := range s.NameCardList {
		if v == nameCardId {
			return
		}
	}
	s.NameCardList = append(s.NameCardList, nameCardId)
}

func (s *DbSocial) UseNameCard(nameCardId uint32) bool {
	exist := false
	for _, v := range s.NameCardList {
		if v == nameCardId {
			exist = true
		}
	}
	if !exist {
		return false
	}
	s.NameCard = nameCardId
	return true
}

func (s *DbSocial) AddFriend(uid uint32) {
	s.FriendList[uid] = uint32(time.Now().Unix())
}

func (s *DbSocial) AddFriendApply(uid uint32) {
	s.FriendApplyList[uid] = uint32(time.Now().Unix())
}

func (s *DbSocial) DelFriend(uid uint32) {
	delete(s.FriendList, uid)
}

func (s *DbSocial) DelFriendApply(uid uint32) {
	delete(s.FriendApplyList, uid)
}

func (s *DbSocial) IsFriend(uid uint32) bool {
	_, exist := s.FriendList[uid]
	return exist
}
