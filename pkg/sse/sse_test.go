package sse

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
	"github.com/tmaxmax/go-sse"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/testing/protocmp"

	pb "github.com/cfhn/our-space/pkg/sse/proto"
)

func TestMapQueryValuesToMessage(t *testing.T) {
	tests := []struct {
		name     string
		values   url.Values
		expected *pb.TestRequest
		wantErr  error
	}{
		{
			name:     "empty",
			values:   url.Values{},
			expected: &pb.TestRequest{},
		},
		{
			name:     "single value",
			values:   url.Values{"string_value": []string{"test"}},
			expected: &pb.TestRequest{StringValue: "test"},
		},
		{
			name: "all possible types, single value",
			values: url.Values{
				"float_value":    []string{"3.14"},
				"double_value":   []string{"3.14159"},
				"int64_value":    []string{"-1"},
				"uint64_value":   []string{"2"},
				"int32_value":    []string{"-3"},
				"fixed64_value":  []string{"4"},
				"fixed32_value":  []string{"5"},
				"bool_value":     []string{"true"},
				"string_value":   []string{"string_value"},
				"bytes_value":    []string{"bytes_value"},
				"uint32_value":   []string{"6"},
				"enum_value":     []string{"ENUM_B"},
				"sfixed32_value": []string{"-7"},
				"sfixed64_value": []string{"-8"},
				"sint32_value":   []string{"-9"},
				"sint64_value":   []string{"-10"},
			},
			expected: &pb.TestRequest{
				FloatValue:    3.14,
				DoubleValue:   3.14159,
				Int64Value:    int64(-1),
				Uint64Value:   uint64(2),
				Int32Value:    int32(-3),
				Fixed64Value:  uint64(4),
				Fixed32Value:  uint32(5),
				BoolValue:     true,
				StringValue:   "string_value",
				BytesValue:    []byte("bytes_value"),
				Uint32Value:   uint32(6),
				EnumValue:     pb.Enum_ENUM_B,
				Sfixed32Value: int32(-7),
				Sfixed64Value: int64(-8),
				Sint32Value:   int32(-9),
				Sint64Value:   int64(-10),
			},
		},
		{
			name: "all possible types, repeated, single value",
			values: url.Values{
				"repeated_float_value":    []string{"3.14"},
				"repeated_double_value":   []string{"3.14159"},
				"repeated_int64_value":    []string{"-1"},
				"repeated_uint64_value":   []string{"2"},
				"repeated_int32_value":    []string{"-3"},
				"repeated_fixed64_value":  []string{"4"},
				"repeated_fixed32_value":  []string{"5"},
				"repeated_bool_value":     []string{"true"},
				"repeated_string_value":   []string{"string_value"},
				"repeated_bytes_value":    []string{"bytes_value"},
				"repeated_uint32_value":   []string{"6"},
				"repeated_enum_value":     []string{"ENUM_B"},
				"repeated_sfixed32_value": []string{"-7"},
				"repeated_sfixed64_value": []string{"-8"},
				"repeated_sint32_value":   []string{"-9"},
				"repeated_sint64_value":   []string{"-10"},
			},
			expected: &pb.TestRequest{
				RepeatedFloatValue:    []float32{3.14},
				RepeatedDoubleValue:   []float64{3.14159},
				RepeatedInt64Value:    []int64{-1},
				RepeatedUint64Value:   []uint64{2},
				RepeatedInt32Value:    []int32{-3},
				RepeatedFixed64Value:  []uint64{4},
				RepeatedFixed32Value:  []uint32{5},
				RepeatedBoolValue:     []bool{true},
				RepeatedStringValue:   []string{"string_value"},
				RepeatedBytesValue:    [][]byte{[]byte("bytes_value")},
				RepeatedUint32Value:   []uint32{6},
				RepeatedEnumValue:     []pb.Enum{pb.Enum_ENUM_B},
				RepeatedSfixed32Value: []int32{-7},
				RepeatedSfixed64Value: []int64{-8},
				RepeatedSint32Value:   []int32{-9},
				RepeatedSint64Value:   []int64{-10},
			},
		},
		{
			name: "all possible types, repeated, multi value",
			values: url.Values{
				"repeated_float_value":    []string{"3.14", "3.141"},
				"repeated_double_value":   []string{"3.14159", "3.141592"},
				"repeated_int64_value":    []string{"-1", "-11"},
				"repeated_uint64_value":   []string{"2", "22"},
				"repeated_int32_value":    []string{"-3", "-33"},
				"repeated_fixed64_value":  []string{"4", "44"},
				"repeated_fixed32_value":  []string{"5", "55"},
				"repeated_bool_value":     []string{"true", "false"},
				"repeated_string_value":   []string{"string_value", "other_string_value"},
				"repeated_bytes_value":    []string{"bytes_value", "other_bytes_value"},
				"repeated_uint32_value":   []string{"6", "66"},
				"repeated_enum_value":     []string{"ENUM_B", "10"},
				"repeated_sfixed32_value": []string{"-7", "-77"},
				"repeated_sfixed64_value": []string{"-8", "-88"},
				"repeated_sint32_value":   []string{"-9", "-99"},
				"repeated_sint64_value":   []string{"-10", "-1010"},
			},
			expected: &pb.TestRequest{
				RepeatedFloatValue:    []float32{3.14, 3.141},
				RepeatedDoubleValue:   []float64{3.14159, 3.141592},
				RepeatedInt64Value:    []int64{-1, -11},
				RepeatedUint64Value:   []uint64{2, 22},
				RepeatedInt32Value:    []int32{-3, -33},
				RepeatedFixed64Value:  []uint64{4, 44},
				RepeatedFixed32Value:  []uint32{5, 55},
				RepeatedBoolValue:     []bool{true, false},
				RepeatedStringValue:   []string{"string_value", "other_string_value"},
				RepeatedBytesValue:    [][]byte{[]byte("bytes_value"), []byte("other_bytes_value")},
				RepeatedUint32Value:   []uint32{6, 66},
				RepeatedEnumValue:     []pb.Enum{pb.Enum_ENUM_B, pb.Enum_ENUM_C},
				RepeatedSfixed32Value: []int32{-7, -77},
				RepeatedSfixed64Value: []int64{-8, -88},
				RepeatedSint32Value:   []int32{-9, -99},
				RepeatedSint64Value:   []int64{-10, -1010},
			},
		},
		{
			name: "json name handling",
			values: url.Values{
				"json_name_handling": []string{"test"},
			},
			expected: &pb.TestRequest{
				Jsonnamehandling: "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &pb.TestRequest{}
			err := mapQueryValuesToMessage(tt.values, req.ProtoReflect())
			require.ErrorIs(t, err, tt.wantErr)
			require.Empty(t, cmp.Diff(tt.expected, req, protocmp.Transform()))
		})
	}
}

type testServiceImpl struct {
	lastRequest *pb.TestRequest
	messages    []*pb.TestResponse

	pb.UnimplementedTestServiceServer
}

func (t *testServiceImpl) Test(req *pb.TestRequest, stream grpc.ServerStreamingServer[pb.TestResponse]) error {
	t.lastRequest = req

	err := stream.SendHeader(metadata.MD{
		"x-test-header": []string{"test", "hello", "world"},
	})
	if err != nil {
		return err
	}

	for _, msg := range t.messages {
		err := stream.Send(msg)
		if err != nil {
			return err
		}
	}

	return nil
}

func TestSseGrpc(t *testing.T) {
	grpcListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	httpListener, err := net.Listen("tcp", "127.0.0.1:0")

	grpcServer := grpc.NewServer()
	testService := &testServiceImpl{
		messages: []*pb.TestResponse{
			{SomeField: "1"},
			{SomeField: "2"},
			{SomeField: "3"},
			{SomeField: "4"},
			{SomeField: "5"},
			{SomeField: "6"},
		},
	}
	pb.RegisterTestServiceServer(grpcServer, testService)

	mux := runtime.NewServeMux()

	grpcClient, err := grpc.NewClient(grpcListener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	err = mux.HandlePath(http.MethodGet, "/test", GrpcProxy[*pb.TestRequest, *pb.TestResponse](grpcClient, pb.TestService_ServiceDesc.Streams[0], pb.TestService_Test_FullMethodName))
	require.NoError(t, err)

	httpServer := http.Server{Handler: mux}

	eg := errgroup.Group{}
	eg.Go(func() error {
		return grpcServer.Serve(grpcListener)
	})
	eg.Go(func() error {
		err := httpServer.Serve(httpListener)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+httpListener.Addr().String()+"/test?bool_value=true&repeated_string_value=Hello&repeated_string_value=World", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	require.ElementsMatch(t, []string{"test", "hello", "world"}, resp.Header.Values("x-test-header"))

	i := 0
	for ev, err := range sse.Read(resp.Body, nil) {
		require.NoError(t, err)

		require.Equal(t, "data", ev.Type)
		require.Equal(t, fmt.Sprintf(`{"some_field":"%d"}`, i+1), ev.Data)
		i++
	}

	require.NoError(t, httpServer.Shutdown(t.Context()))
	grpcServer.Stop()

	err = eg.Wait()
	require.NoError(t, err)
}
