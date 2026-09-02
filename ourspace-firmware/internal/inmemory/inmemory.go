package inmemory

import (
	"bytes"
	"sync"
	"sync/atomic"

	"google.golang.org/protobuf/proto"

	pbBackend "github.com/cfhn/our-space/ourspace-backend/proto"
	pb "github.com/cfhn/our-space/ourspace-firmware/proto"
)

type Repository struct {
	members        atomic.Pointer[map[string]*pbBackend.Member]
	cards          atomic.Pointer[map[string]*pbBackend.Card]
	localPresences []*pb.LocalPresence

	mu sync.RWMutex
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
		if bytes.Equal(card.RfidValue, rfidValue) {
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

func (r *Repository) CreatePresence(presence *pb.LocalPresence) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.localPresences = append(r.localPresences, presence)
}

func (r *Repository) UpdatePresence(presence *pb.LocalPresence) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, p := range r.localPresences {
		if p.Presence.Id == presence.Presence.Id {
			r.localPresences[i] = presence
			break
		}
	}
}

func (r *Repository) ListPresences() []*pb.LocalPresence {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*pb.LocalPresence, 0, len(r.localPresences))
	for _, p := range r.localPresences {
		result = append(result, proto.CloneOf(p))
	}

	return result
}

func (r *Repository) FindActivePresence(memberId string) *pb.LocalPresence {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.localPresences {
		if p.Presence.MemberId == memberId && p.Presence.CheckoutTime == nil {
			return proto.CloneOf(p)
		}
	}

	return nil
}

func (r *Repository) DeletePresence(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, p := range r.localPresences {
		if p.Presence.Id == id {
			r.localPresences = append(r.localPresences[:i], r.localPresences[i+1:]...)
			break
		}
	}
}
