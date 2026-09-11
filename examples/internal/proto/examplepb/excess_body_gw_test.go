package examplepb_test

import (
	"context"
	"io"
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

type bodylessEchoServer struct {
	examplepb.UnimplementedEchoServiceServer
	called bool
}

func (s *bodylessEchoServer) Echo(_ context.Context, req *examplepb.SimpleMessage) (*examplepb.SimpleMessage, error) {
	s.called = true
	return req, nil
}

func (s *bodylessEchoServer) EchoDelete(_ context.Context, req *examplepb.SimpleMessage) (*examplepb.SimpleMessage, error) {
	s.called = true
	return req, nil
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

func TestGeneratedBodylessHandlerRejectsBody(t *testing.T) {
	for _, spec := range []struct {
		name           string
		method         string
		target         string
		withBody       bool
		contentType    string
		methodOverride string
		wantStatus     int
		wantCalled     bool
		wantBody       string
	}{
		{
			name:       "POST allows empty body",
			method:     http.MethodPost,
			target:     "/v1/example/echo/id",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "POST rejects body",
			method:     http.MethodPost,
			target:     "/v1/example/echo/id",
			withBody:   true,
			wantStatus: http.StatusBadRequest,
			wantBody:   "request body is not allowed for this HTTP binding",
		},
		{
			name:       "DELETE allows empty body",
			method:     http.MethodDelete,
			target:     "/v1/example/echo_delete",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "DELETE rejects body",
			method:     http.MethodDelete,
			target:     "/v1/example/echo_delete",
			withBody:   true,
			wantStatus: http.StatusBadRequest,
			wantBody:   "request body is not allowed for this HTTP binding",
		},
		{
			name:       "GET rejects body",
			method:     http.MethodGet,
			target:     "/v1/example/echo/id/1",
			withBody:   true,
			wantStatus: http.StatusBadRequest,
			wantBody:   "request body is not allowed for this HTTP binding",
		},
		{
			name:        "POST-to-GET fallback allows form body",
			method:      http.MethodPost,
			target:      "/v1/example/echo/id/1",
			withBody:    true,
			contentType: "application/x-www-form-urlencoded",
			wantStatus:  http.StatusOK,
			wantCalled:  true,
		},
		{
			name:           "POST-to-GET method override allows form body",
			method:         http.MethodPost,
			target:         "/v1/example/echo/id/1",
			withBody:       true,
			contentType:    "application/x-www-form-urlencoded",
			methodOverride: http.MethodGet,
			wantStatus:     http.StatusOK,
			wantCalled:     true,
		},
	} {
		t.Run(spec.name, func(t *testing.T) {
			mux := runtime.NewServeMux()
			server := &bodylessEchoServer{}
			if err := examplepb.RegisterEchoServiceHandlerServer(t.Context(), mux, server); err != nil {
				t.Fatalf("RegisterEchoServiceHandlerServer() failed: %v", err)
			}

			var body io.Reader
			if spec.withBody {
				body = strings.NewReader("value=body")
			}
			req := httptest.NewRequest(spec.method, spec.target, body)
			if spec.contentType != "" {
				req.Header.Set("Content-Type", spec.contentType)
			}
			if spec.methodOverride != "" {
				req.Header.Set("X-HTTP-Method-Override", spec.methodOverride)
			}
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)

			if got := resp.Code; got != spec.wantStatus {
				t.Errorf("status = %d; want %d; body = %s", got, spec.wantStatus, resp.Body.String())
			}
			if server.called != spec.wantCalled {
				t.Errorf("service called = %v; want %v", server.called, spec.wantCalled)
			}
			if got := resp.Body.String(); !strings.Contains(got, spec.wantBody) {
				t.Errorf("body = %q; want to contain %q", got, spec.wantBody)
			}
		})
	}
}
