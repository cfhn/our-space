package presences

import (
	"context"
	"github.com/cfhn/our-space/ourspace-backend/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service struct {
	repo *Postgres
	// May need mocking similiar to cards service
	memberService pb.MemberServiceServer
	pb.UnimplementedPresenceServiceServer
}

func NewService(repo *Postgres, member pb.MemberServiceServer) *Service {
	return &Service{repo: repo, memberService: member}
}

// mustEmbedUnimplementedPresenceServiceServer implements pb.PresenceServiceServer.
func (s *Service) mustEmbedUnimplementedPresenceServiceServer() {
	panic("unimplemented")
}

// Checkin implements pb.PresenceServiceServer.
func (s *Service) Checkin(context.Context, *pb.CheckinRequest) (*pb.Presence, error) {
	panic("unimplemented")
}

// Checkout implements pb.PresenceServiceServer.
func (s *Service) Checkout(context.Context, *pb.CheckoutRequest) (*pb.Presence, error) {
	panic("unimplemented")
}

func (s *Service) DeletePresence(ctx context.Context, request *pb.DeletePresenceRequest) (*emptypb.Empty, error) {
	if err := s.repo.DeletePresence(ctx, request.Id); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// ListPresences implements pb.PresenceServiceServer.
func (s *Service) ListPresences(context.Context, *pb.ListPresencesRequest) (*pb.ListPresencesResponse, error) {
	panic("unimplemented")
}

// UpdatePresence implements pb.PresenceServiceServer.
func (s *Service) UpdatePresence(context.Context, *pb.UpdatePresenceRequest) (*pb.Presence, error) {
	panic("unimplemented")
}
