package examplepb_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/examples/internal/proto/examplepb"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/types/known/emptypb"
)

type excessBodyServer struct {
	examplepb.UnimplementedExcessBodyServiceServer
	called bool
}

func (s *excessBodyServer) NoBodyRpc(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	s.called = true
	return &emptypb.Empty{}, nil
}

func TestGeneratedQuerylessHandlerInvokesQueryParser(t *testing.T) {
	defer runtime.NewServeMux(runtime.SetQueryParameterParser(&runtime.DefaultQueryParser{}))

	for _, spec := range []struct {
		name       string
		muxOptions []runtime.ServeMuxOption
		wantStatus int
		wantCalled bool
		wantBody   string
	}{
		{
			name:       "default parser ignores unknown parameter",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name: "strict parser rejects unknown parameter",
			muxOptions: []runtime.ServeMuxOption{runtime.SetQueryParameterParser(
				&runtime.DefaultQueryParser{RejectUnknownFields: true},
			)},
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
			wantBody:   `unknown query parameter \"unexpected\"`,
		},
	} {
		t.Run(spec.name, func(t *testing.T) {
			mux := runtime.NewServeMux(spec.muxOptions...)
			server := &excessBodyServer{}
			if err := examplepb.RegisterExcessBodyServiceHandlerServer(t.Context(), mux, server); err != nil {
				t.Fatalf("RegisterExcessBodyServiceHandlerServer() failed: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/rpc/excess-body/rpc?unexpected=value", nil)
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)

			if got := resp.Code; got != spec.wantStatus {
				t.Errorf("status = %d; want %d; body = %s", got, spec.wantStatus, resp.Body.String())
			}
			if got := resp.Body.String(); !strings.Contains(got, spec.wantBody) {
				t.Errorf("body = %q; want to contain %q", got, spec.wantBody)
			}
			if server.called != spec.wantCalled {
				t.Errorf("service called = %v; want %v", server.called, spec.wantCalled)
			}
		})
	}
}
