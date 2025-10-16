package inmemory

import (
	"sync/atomic"

	pbBackend "github.com/cfhn/our-space/ourspace-backend/proto"
)

type Repository struct {
	members atomic.Pointer[map[string]*pbBackend.Member]
	cards   atomic.Pointer[map[string]*pbBackend.Card]
}

func NewRepository() *Repository {
	return &Repository{
		members: atomic.Pointer[map[string]*pbBackend.Member]{},
		cards:   atomic.Pointer[map[string]*pbBackend.Card]{},
	}
}

func (r *Repository) Replace(members []*pbBackend.Member, cards []*pbBackend.Card) {
	memberMap := make(map[string]*pbBackend.Member, len(members))

	for _, member := range members {
		memberMap[member.Id] = member
	}

	cardMap := make(map[string]*pbBackend.Card, len(cards))

	for _, card := range cards {
		cardMap[card.Id] = card
	}

	r.members.Store(&memberMap)
	r.cards.Store(&cardMap)
}

func (r *Repository) FindCardByRFID(rfidValue []byte) *pbBackend.Card {
	cards := r.cards.Load()
	if cards == nil {
		return nil
	}

	for _, card := range *cards {
		if string(card.RfidValue) == string(rfidValue) {
			return card
		}
	}

	return nil
}

func (r *Repository) FindMemberByID(id string) *pbBackend.Member {
	members := r.members.Load()
	if members == nil {
		return nil
	}

	return (*members)[id]
}
