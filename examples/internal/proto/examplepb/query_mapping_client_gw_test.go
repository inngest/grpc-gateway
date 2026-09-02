package examplepb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type queryMappingClient struct {
	ABitOfEverythingServiceClient
	request *ABitOfEverything
}

func (c *queryMappingClient) CreateBody(_ context.Context, req *ABitOfEverything, _ ...grpc.CallOption) (*ABitOfEverything, error) {
	c.request = req
	return req, nil
}

func TestGeneratedClientHandlerDoesNotOverwriteWildcardBodyFromQuery(t *testing.T) {
	defer runtime.NewServeMux(runtime.SetQueryParameterParser(&runtime.DefaultQueryParser{}))

	for _, spec := range []struct {
		name   string
		strict bool
	}{
		{name: "permissive parser"},
		{name: "strict parser", strict: true},
	} {
		t.Run(spec.name, func(t *testing.T) {
			runtime.NewServeMux(runtime.SetQueryParameterParser(
				&runtime.DefaultQueryParser{RejectUnknownFields: spec.strict},
			))
			client := &queryMappingClient{}
			req := httptest.NewRequest(
				http.MethodPost,
				"/v1/example/a_bit_of_everything?string_value=query/event",
				strings.NewReader(`{"stringValue":"body/event"}`),
			)
			_, _, err := request_ABitOfEverythingService_CreateBody_0(
				t.Context(),
				&runtime.JSONPb{},
				client,
				req,
				nil,
			)

			if spec.strict {
				if got := status.Code(err); got != codes.InvalidArgument {
					t.Fatalf("status.Code(error) = %v; want %v; error = %v", got, codes.InvalidArgument, err)
				}
				if want := `query parameter "string_value" is mapped to the request body or path and cannot be set in the query string`; !strings.Contains(err.Error(), want) {
					t.Errorf("error = %q; want to contain %q", err, want)
				}
				if client.request != nil {
					t.Fatal("CreateBody() was called; want query parameter rejected first")
				}
				return
			}

			if err != nil {
				t.Fatalf("request handler failed: %v", err)
			}
			if got := client.request.GetStringValue(); got != "body/event" {
				t.Errorf("CreateBody() string_value = %q; want body/event", got)
			}
		})
	}
}
