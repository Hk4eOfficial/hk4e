package model

type Team struct {
	Name         string
	AvatarIdList []uint32
}

func (t *Team) GetAvatarIdList() []uint32 {
	avatarIdList := make([]uint32, 0)
	for _, avatarId := range t.AvatarIdList {
		if avatarId == 0 {
			continue
		}
		avatarIdList = append(avatarIdList, avatarId)
	}
	return avatarIdList
}

func (t *Team) SetAvatarIdList(avatarIdList []uint32) {
	t.AvatarIdList = make([]uint32, 4)
	for index := range t.AvatarIdList {
		if index >= len(avatarIdList) {
			break
		}
		t.AvatarIdList[index] = avatarIdList[index]
	}
}

type DbTeam struct {
	TeamList             []*Team
	CurrTeamIndex        uint8
	CurrAvatarIndex      uint8
	TeamResonances       map[uint16]bool `bson:"-" msgpack:"-"`
	TeamResonancesConfig map[int32]bool  `bson:"-" msgpack:"-"`
}

func (p *Player) GetDbTeam() *DbTeam {
	if p.DbTeam == nil {
		p.DbTeam = new(DbTeam)
	}
	if p.DbTeam.TeamList == nil {
		p.DbTeam.TeamList = []*Team{
			{Name: "", AvatarIdList: make([]uint32, 4)},
			{Name: "", AvatarIdList: make([]uint32, 4)},
			{Name: "", AvatarIdList: make([]uint32, 4)},
			{Name: "", AvatarIdList: make([]uint32, 4)},
		}
	}
	if p.DbTeam.CurrTeamIndex == 0 {
		p.DbTeam.CurrTeamIndex = 0
	}
	if p.DbTeam.CurrAvatarIndex == 0 {
		p.DbTeam.CurrAvatarIndex = 0
	}
	return p.DbTeam
}

func (t *DbTeam) GetActiveTeamId() uint8 {
	return t.CurrTeamIndex + 1
}

func (t *DbTeam) GetTeamByIndex(teamIndex uint8) *Team {
	if t.TeamList == nil {
		return nil
	}
	if teamIndex >= uint8(len(t.TeamList)) {
		return nil
	}
	activeTeam := t.TeamList[teamIndex]
	return activeTeam
}

func (t *DbTeam) GetActiveTeam() *Team {
	return t.GetTeamByIndex(t.CurrTeamIndex)
}

func (t *DbTeam) GetActiveAvatarId() uint32 {
	team := t.GetActiveTeam()
	if team == nil {
		return 0
	}
	return team.AvatarIdList[t.CurrAvatarIndex]
}
