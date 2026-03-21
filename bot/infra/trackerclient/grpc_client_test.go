package trackerclient

import (
	"context"
	"errors"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/adapters/dto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeConn struct {
	invokeFunc func(ctx context.Context, method string, args any, reply any, opts ...grpc.CallOption) error
	closeErr   error
}

func (f *fakeConn) Invoke(ctx context.Context, method string, args any, reply any, opts ...grpc.CallOption) error {
	if f.invokeFunc != nil {
		return f.invokeFunc(ctx, method, args, reply, opts...)
	}
	return nil
}

func (f *fakeConn) Close() error {
	return f.closeErr
}

type methodTestCase struct {
	name   string
	invoke func(t *testing.T, method string, args any, reply any) error
	call   func(ctx context.Context, c *GRPCClient) (any, error)
	assert func(t *testing.T, got any, err error)
}

func runMethodCases(t *testing.T, tests []methodTestCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conn := &fakeConn{
				invokeFunc: func(_ context.Context, method string, args any, reply any, _ ...grpc.CallOption) error {
					return tt.invoke(t, method, args, reply)
				},
			}
			client := &GRPCClient{conn: conn, closer: conn}

			got, err := tt.call(context.Background(), client)
			tt.assert(t, got, err)
		})
	}
}

func TestGRPCClient_Methods_ErrorMapping_NoServer(t *testing.T) {
	t.Parallel()

	runMethodCases(t, []methodTestCase{
		{
			name: "register chat maps not found",
			invoke: func(t *testing.T, method string, _ any, _ any) error {
				t.Helper()
				if method != "/scrapper.ScrapperService/RegisterChat" {
					t.Fatalf("unexpected method: %s", method)
				}
				return status.Error(codes.NotFound, "chat missing")
			},
			call: func(ctx context.Context, c *GRPCClient) (any, error) {
				return nil, c.RegisterChat(ctx, 1)
			},
			assert: func(t *testing.T, _ any, err error) {
				t.Helper()
				if !errors.Is(err, tracker.ErrNotFound) {
					t.Fatalf("expected tracker.ErrNotFound, got: %v", err)
				}
			},
		},
		{
			name: "delete chat maps invalid argument",
			invoke: func(t *testing.T, method string, _ any, _ any) error {
				t.Helper()
				if method != "/scrapper.ScrapperService/DeleteChat" {
					t.Fatalf("unexpected method: %s", method)
				}
				return status.Error(codes.InvalidArgument, "bad id")
			},
			call: func(ctx context.Context, c *GRPCClient) (any, error) {
				return nil, c.DeleteChat(ctx, 0)
			},
			assert: func(t *testing.T, _ any, err error) {
				t.Helper()
				if !errors.Is(err, tracker.ErrBadRequest) {
					t.Fatalf("expected tracker.ErrBadRequest, got: %v", err)
				}
			},
		},
	})
}

func TestGRPCClient_Methods_Success_NoServer(t *testing.T) {
	t.Parallel()

	runMethodCases(t, []methodTestCase{
		addLinkSuccessCase(),
		removeLinkSuccessCase(),
		listLinksSuccessCase(),
	})
}

func addLinkSuccessCase() methodTestCase {
	return methodTestCase{
		name: "add link success",
		invoke: func(t *testing.T, method string, args any, reply any) error {
			t.Helper()
			if method != "/scrapper.ScrapperService/AddLink" {
				t.Fatalf("unexpected method: %s", method)
			}
			req, ok := args.(dto.AddLinkGRPCRequest)
			if !ok {
				t.Fatalf("unexpected request type: %T", args)
			}
			resp, ok := reply.(*dto.AddLinkGRPCResponse)
			if !ok {
				t.Fatalf("unexpected response type: %T", reply)
			}
			resp.Link = api.LinkResponse{ID: 10, URL: req.Body.Link}
			return nil
		},
		call: func(ctx context.Context, c *GRPCClient) (any, error) {
			return c.AddLink(ctx, 1, api.AddLinkRequest{Link: "https://github.com/user/repo"})
		},
		assert: func(t *testing.T, got any, err error) {
			t.Helper()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			link, ok := got.(api.LinkResponse)
			if !ok {
				t.Fatalf("unexpected result type: %T", got)
			}
			if link.ID != 10 || link.URL != "https://github.com/user/repo" {
				t.Fatalf("unexpected response: %+v", link)
			}
		},
	}
}

func removeLinkSuccessCase() methodTestCase {
	return methodTestCase{
		name: "remove link success",
		invoke: func(t *testing.T, method string, args any, reply any) error {
			t.Helper()
			if method != "/scrapper.ScrapperService/RemoveLink" {
				t.Fatalf("unexpected method: %s", method)
			}
			req, ok := args.(dto.RemoveLinkGRPCRequest)
			if !ok {
				t.Fatalf("unexpected request type: %T", args)
			}
			resp, ok := reply.(*dto.RemoveLinkGRPCResponse)
			if !ok {
				t.Fatalf("unexpected response type: %T", reply)
			}
			resp.Link = api.LinkResponse{ID: 11, URL: req.Body.Link}
			return nil
		},
		call: func(ctx context.Context, c *GRPCClient) (any, error) {
			return c.RemoveLink(ctx, 1, api.RemoveLinkRequest{Link: "https://github.com/user/repo"})
		},
		assert: func(t *testing.T, got any, err error) {
			t.Helper()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			link, ok := got.(api.LinkResponse)
			if !ok {
				t.Fatalf("unexpected result type: %T", got)
			}
			if link.ID != 11 || link.URL != "https://github.com/user/repo" {
				t.Fatalf("unexpected response: %+v", link)
			}
		},
	}
}

func listLinksSuccessCase() methodTestCase {
	return methodTestCase{
		name: "list links success",
		invoke: func(t *testing.T, method string, args any, reply any) error {
			t.Helper()
			if method != "/scrapper.ScrapperService/ListLinks" {
				t.Fatalf("unexpected method: %s", method)
			}
			_, ok := args.(dto.ListLinksRequest)
			if !ok {
				t.Fatalf("unexpected request type: %T", args)
			}
			resp, ok := reply.(*dto.ListLinksResponse)
			if !ok {
				t.Fatalf("unexpected response type: %T", reply)
			}
			resp.Body = api.ListLinksResponse{
				Links: []api.LinkResponse{{ID: 42, URL: "https://github.com/user/repo"}},
				Size:  1,
			}
			return nil
		},
		call: func(ctx context.Context, c *GRPCClient) (any, error) {
			return c.ListLinks(ctx, 1)
		},
		assert: func(t *testing.T, got any, err error) {
			t.Helper()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			resp, ok := got.(api.ListLinksResponse)
			if !ok {
				t.Fatalf("unexpected result type: %T", got)
			}
			if resp.Size != 1 || len(resp.Links) != 1 || resp.Links[0].ID != 42 {
				t.Fatalf("unexpected list response: %+v", resp)
			}
		},
	}
}

func TestGRPCClient_Close(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		closer  grpcCloser
		wantErr bool
	}{
		{name: "nil closer", closer: nil, wantErr: false},
		{name: "close success", closer: &fakeConn{}, wantErr: false},
		{name: "close wraps error", closer: &fakeConn{closeErr: errors.New("close failed")}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &GRPCClient{closer: tt.closer}
			err := client.Close()
			if tt.wantErr && err == nil {
				t.Fatalf("expected close error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected close error: %v", err)
			}
		})
	}
}

func TestMapGRPCError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{name: "nil", err: nil, wantErr: nil},
		{name: "not found", err: status.Error(codes.NotFound, "x"), wantErr: tracker.ErrNotFound},
		{name: "already exists", err: status.Error(codes.AlreadyExists, "x"), wantErr: tracker.ErrAlreadyExists},
		{name: "invalid argument", err: status.Error(codes.InvalidArgument, "x"), wantErr: tracker.ErrBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := mapGRPCError(tt.err)
			if tt.wantErr == nil {
				if got != nil {
					t.Fatalf("expected nil, got: %v", got)
				}
				return
			}
			if !errors.Is(got, tt.wantErr) {
				t.Fatalf("expected %v, got: %v", tt.wantErr, got)
			}
		})
	}
}
