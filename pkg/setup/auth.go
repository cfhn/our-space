package setup

import (
	"context"
	"crypto/ecdsa"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	pb "github.com/cfhn/our-space/pkg/setup/proto"
	"github.com/cfhn/our-space/pkg/status"
)

type AccessTokenClaims struct {
	jwt.RegisteredClaims
	Type     string `json:"type"`
	FullName string `json:"full_name"`
}

type RefreshTokenClaims struct {
	jwt.RegisteredClaims
	Type      string           `json:"type"`
	LoginTime *jwt.NumericDate `json:"login_time"`
}

type accessTokenClaimsKey struct{}

func AuthInterceptor(keyFunc func(kid string) *ecdsa.PublicKey) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (resp any, err error) {
		if !shouldAuthenticate(fullMethodToMethodName(info.FullMethod)) {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Unauthenticated()
		}

		authorizationHeaders := md.Get("authorization")
		if len(authorizationHeaders) != 1 {
			return nil, status.Unauthenticated()
		}

		authorizationHeader := authorizationHeaders[0]

		token, ok := strings.CutPrefix(authorizationHeader, "Bearer ")
		if !ok {
			return nil, status.Unauthenticated()
		}

		var accessTokenClaims AccessTokenClaims
		_, err = jwt.ParseWithClaims(token, &accessTokenClaims, func(token *jwt.Token) (any, error) {
			kid, ok := token.Header["kid"].(string)
			if !ok {
				return nil, nil
			}

			return keyFunc(kid), nil
		}, jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{jwt.SigningMethodES256.Name}))
		if err != nil {
			return nil, status.PermissionDenied()
		}

		if accessTokenClaims.Type != "access" {
			return nil, status.PermissionDenied()
		}

		reqCtx := context.WithValue(ctx, accessTokenClaimsKey{}, accessTokenClaims)

		return handler(reqCtx, req)
	}
}

func GetAccessTokenClaims(ctx context.Context) (*AccessTokenClaims, bool) {
	v := ctx.Value(accessTokenClaimsKey{})
	if v == nil {
		return nil, false
	}

	claims, ok := v.(AccessTokenClaims)
	if !ok {
		return nil, false
	}

	return &claims, true
}

func fullMethodToMethodName(fullMethod string) string {
	return strings.Replace(strings.TrimPrefix(fullMethod, "/"), "/", ".", 1)
}

func shouldAuthenticate(methodName string) bool {
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(methodName))
	if err != nil {
		return true
	}

	methodOptions := desc.Options().(*descriptorpb.MethodOptions)
	authOptions := proto.GetExtension(methodOptions, pb.E_AuthOptions).(*pb.AuthOptions)

	if authOptions == nil {
		return true
	}

	return !authOptions.AllowUnauthenticated
}
