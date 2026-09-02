package examplepb_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/examples/internal/proto/examplepb"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type queryMappingServer struct {
	examplepb.UnimplementedABitOfEverythingServiceServer
	createBodyRequest *examplepb.ABitOfEverything
	createBookRequest *examplepb.CreateBookRequest
}

func (s *queryMappingServer) CreateBody(_ context.Context, req *examplepb.ABitOfEverything) (*examplepb.ABitOfEverything, error) {
	s.createBodyRequest = req
	return req, nil
}

func (s *queryMappingServer) CreateBook(_ context.Context, req *examplepb.CreateBookRequest) (*examplepb.Book, error) {
	s.createBookRequest = req
	return req.Book, nil
}

func TestGeneratedLocalHandlerDoesNotOverwriteWildcardBodyFromQuery(t *testing.T) {
	defer runtime.NewServeMux(runtime.SetQueryParameterParser(&runtime.DefaultQueryParser{}))

	for _, spec := range []struct {
		name       string
		strict     bool
		wantStatus int
		wantError  string
	}{
		{
			name:       "permissive parser ignores query field",
			wantStatus: http.StatusOK,
		},
		{
			name:       "strict parser rejects query field",
			strict:     true,
			wantStatus: http.StatusBadRequest,
			wantError:  `query parameter \"string_value\" is mapped to the request body or path and cannot be set in the query string`,
		},
	} {
		t.Run(spec.name, func(t *testing.T) {
			server := &queryMappingServer{}
			mux := newQueryMappingMux(t, server, spec.strict)
			req := httptest.NewRequest(
				http.MethodPost,
				"/v1/example/a_bit_of_everything?string_value=query/event",
				strings.NewReader(`{"stringValue":"body/event"}`),
			)
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)

			if got := resp.Code; got != spec.wantStatus {
				t.Fatalf("status = %d; want %d; body = %s", got, spec.wantStatus, resp.Body.String())
			}
			if !strings.Contains(resp.Body.String(), spec.wantError) {
				t.Errorf("body = %q; want to contain %q", resp.Body.String(), spec.wantError)
			}
			if spec.strict {
				if server.createBodyRequest != nil {
					t.Fatal("CreateBody() was called; want query parameter rejected first")
				}
			} else if got := server.createBodyRequest.GetStringValue(); got != "body/event" {
				t.Errorf("CreateBody() string_value = %q; want body/event", got)
			}
		})
	}
}

func TestGeneratedLocalHandlerRejectsBodyAndPathFieldsFromQuery(t *testing.T) {
	defer runtime.NewServeMux(runtime.SetQueryParameterParser(&runtime.DefaultQueryParser{}))

	for _, spec := range []struct {
		name      string
		target    string
		strict    bool
		wantError string
	}{
		{
			name:   "permissive parser preserves body and path",
			target: "/v1/publishers/path/books?book.name=query/book&parent=publishers/query",
		},
		{
			name:      "strict parser rejects body descendant",
			target:    "/v1/publishers/path/books?book.name=query/book",
			strict:    true,
			wantError: `query parameter \"book.name\" is mapped to the request body or path and cannot be set in the query string`,
		},
		{
			name:      "strict parser rejects path field",
			target:    "/v1/publishers/path/books?parent=publishers/query",
			strict:    true,
			wantError: `query parameter \"parent\" is mapped to the request body or path and cannot be set in the query string`,
		},
	} {
		t.Run(spec.name, func(t *testing.T) {
			server := &queryMappingServer{}
			mux := newQueryMappingMux(t, server, spec.strict)
			req := httptest.NewRequest(http.MethodPost, spec.target, strings.NewReader(`{"name":"body/book"}`))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)

			if spec.strict {
				if got := resp.Code; got != http.StatusBadRequest {
					t.Fatalf("status = %d; want %d; body = %s", got, http.StatusBadRequest, resp.Body.String())
				}
				if !strings.Contains(resp.Body.String(), spec.wantError) {
					t.Errorf("body = %q; want to contain %q", resp.Body.String(), spec.wantError)
				}
				if server.createBookRequest != nil {
					t.Fatal("CreateBook() was called; want query parameter rejected first")
				}
				return
			}

			if got := resp.Code; got != http.StatusOK {
				t.Fatalf("status = %d; want %d; body = %s", got, http.StatusOK, resp.Body.String())
			}
			if got := server.createBookRequest.GetBook().GetName(); got != "body/book" {
				t.Errorf("CreateBook() book.name = %q; want body/book", got)
			}
			if got := server.createBookRequest.GetParent(); got != "publishers/path" {
				t.Errorf("CreateBook() parent = %q; want publishers/path", got)
			}
		})
	}
}

func newQueryMappingMux(t *testing.T, server *queryMappingServer, strict bool) *runtime.ServeMux {
	t.Helper()
	parser := &runtime.DefaultQueryParser{RejectUnknownFields: strict}
	mux := runtime.NewServeMux(runtime.SetQueryParameterParser(parser))
	if err := examplepb.RegisterABitOfEverythingServiceHandlerServer(t.Context(), mux, server); err != nil {
		t.Fatalf("RegisterABitOfEverythingServiceHandlerServer() failed: %v", err)
	}
	return mux
}
