package status

import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func FieldViolations(fieldViolations []*errdetails.BadRequest_FieldViolation) error {
	errStatus := status.New(codes.InvalidArgument, "validation errors")
	errStatus, err := errStatus.WithDetails(
		&errdetails.BadRequest{FieldViolations: fieldViolations},
	)
	if err != nil {
		return status.Error(codes.Internal, "internal server error")
	}

	return errStatus.Err()
}

func Internal(_ error) error {
	return status.Error(codes.Internal, "internal server error")
}

func Unimplemented() error {
	return status.Error(codes.Unimplemented, "not implemented")
}

func NotFound() error {
	return status.Error(codes.NotFound, "not found")
}

func FromError(err error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "")
	}

	s, ok := status.FromError(err)
	if !ok {
		return status.New(codes.Internal, err.Error())
	}

	return s
}
