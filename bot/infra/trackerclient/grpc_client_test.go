package trackerclient

import (
	"context"
	"errors"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	scrapperv1 "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/grpcapi/scrapperv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeScrapperClient struct {
	registerChatFunc func(context.Context, *scrapperv1.RegisterChatRequest, ...grpc.CallOption) (*scrapperv1.RegisterChatResponse, error)
	deleteChatFunc   func(context.Context, *scrapperv1.DeleteChatRequest, ...grpc.CallOption) (*scrapperv1.DeleteChatResponse, error)
	addLinkFunc      func(context.Context, *scrapperv1.AddLinkRequest, ...grpc.CallOption) (*scrapperv1.AddLinkResponse, error)
	removeLinkFunc   func(context.Context, *scrapperv1.RemoveLinkRequest, ...grpc.CallOption) (*scrapperv1.RemoveLinkResponse, error)
	listLinksFunc    func(context.Context, *scrapperv1.ListLinksRequest, ...grpc.CallOption) (*scrapperv1.ListLinksResponse, error)
}

func (f *fakeScrapperClient) RegisterChat(ctx context.Context, req *scrapperv1.RegisterChatRequest, opts ...grpc.CallOption) (*scrapperv1.RegisterChatResponse, error) {
	if f.registerChatFunc != nil {
		return f.registerChatFunc(ctx, req, opts...)
	}
	return &scrapperv1.RegisterChatResponse{}, nil
}

func (f *fakeScrapperClient) DeleteChat(ctx context.Context, req *scrapperv1.DeleteChatRequest, opts ...grpc.CallOption) (*scrapperv1.DeleteChatResponse, error) {
	if f.deleteChatFunc != nil {
		return f.deleteChatFunc(ctx, req, opts...)
	}
	return &scrapperv1.DeleteChatResponse{}, nil
}

func (f *fakeScrapperClient) AddLink(ctx context.Context, req *scrapperv1.AddLinkRequest, opts ...grpc.CallOption) (*scrapperv1.AddLinkResponse, error) {
	if f.addLinkFunc != nil {
		return f.addLinkFunc(ctx, req, opts...)
	}
	return &scrapperv1.AddLinkResponse{}, nil
}

func (f *fakeScrapperClient) RemoveLink(ctx context.Context, req *scrapperv1.RemoveLinkRequest, opts ...grpc.CallOption) (*scrapperv1.RemoveLinkResponse, error) {
	if f.removeLinkFunc != nil {
		return f.removeLinkFunc(ctx, req, opts...)
	}
	return &scrapperv1.RemoveLinkResponse{}, nil
}

func (f *fakeScrapperClient) ListLinks(ctx context.Context, req *scrapperv1.ListLinksRequest, opts ...grpc.CallOption) (*scrapperv1.ListLinksResponse, error) {
	if f.listLinksFunc != nil {
		return f.listLinksFunc(ctx, req, opts...)
	}
	return &scrapperv1.ListLinksResponse{}, nil
}

type fakeCloser struct {
	closeErr error
}

func (f *fakeCloser) Close() error {
	return f.closeErr
}

func TestGRPCClient_Methods_ErrorMapping_NoServer(t *testing.T) {
	t.Parallel()

	t.Run("register chat maps not found", func(t *testing.T) {
		t.Parallel()
		client := &GRPCClient{
			client: &fakeScrapperClient{
				registerChatFunc: func(_ context.Context, req *scrapperv1.RegisterChatRequest, _ ...grpc.CallOption) (*scrapperv1.RegisterChatResponse, error) {
					if req.GetChatId() != 1 {
						t.Fatalf("unexpected chat id: %d", req.GetChatId())
					}
					return nil, status.Error(codes.NotFound, "chat missing")
				},
			},
		}
		err := client.RegisterChat(context.Background(), 1)
		if !errors.Is(err, tracker.ErrNotFound) {
			t.Fatalf("expected tracker.ErrNotFound, got: %v", err)
		}
	})

	t.Run("delete chat maps invalid argument", func(t *testing.T) {
		t.Parallel()
		client := &GRPCClient{
			client: &fakeScrapperClient{
				deleteChatFunc: func(_ context.Context, req *scrapperv1.DeleteChatRequest, _ ...grpc.CallOption) (*scrapperv1.DeleteChatResponse, error) {
					if req.GetChatId() != 0 {
						t.Fatalf("unexpected chat id: %d", req.GetChatId())
					}
					return nil, status.Error(codes.InvalidArgument, "bad id")
				},
			},
		}
		err := client.DeleteChat(context.Background(), 0)
		if !errors.Is(err, tracker.ErrBadRequest) {
			t.Fatalf("expected tracker.ErrBadRequest, got: %v", err)
		}
	})
}

func TestGRPCClient_Methods_Success_NoServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		call   func(context.Context, *GRPCClient) (any, error)
		assert func(*testing.T, any, error)
		client scrapperGRPCClient
	}{
		addLinkSuccessCase(t),
		removeLinkSuccessCase(t),
		listLinksSuccessCase(t),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := &GRPCClient{client: tt.client}
			got, err := tt.call(context.Background(), client)
			tt.assert(t, got, err)
		})
	}
}

func addLinkSuccessCase(t *testing.T) struct {
	name   string
	call   func(context.Context, *GRPCClient) (any, error)
	assert func(*testing.T, any, error)
	client scrapperGRPCClient
} {
	t.Helper()
	return struct {
		name   string
		call   func(context.Context, *GRPCClient) (any, error)
		assert func(*testing.T, any, error)
		client scrapperGRPCClient
	}{
		name: "add link success",
		client: &fakeScrapperClient{
			addLinkFunc: func(_ context.Context, req *scrapperv1.AddLinkRequest, _ ...grpc.CallOption) (*scrapperv1.AddLinkResponse, error) {
				if req.GetChatId() != 1 {
					t.Fatalf("unexpected chat id: %d", req.GetChatId())
				}
				if req.GetBody().GetLink() != "https://github.com/user/repo" {
					t.Fatalf("unexpected link: %s", req.GetBody().GetLink())
				}
				return &scrapperv1.AddLinkResponse{
					Link: &scrapperv1.LinkResponse{
						Id:  10,
						Url: req.GetBody().GetLink(),
					},
				}, nil
			},
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

func removeLinkSuccessCase(t *testing.T) struct {
	name   string
	call   func(context.Context, *GRPCClient) (any, error)
	assert func(*testing.T, any, error)
	client scrapperGRPCClient
} {
	t.Helper()
	return struct {
		name   string
		call   func(context.Context, *GRPCClient) (any, error)
		assert func(*testing.T, any, error)
		client scrapperGRPCClient
	}{
		name: "remove link success",
		client: &fakeScrapperClient{
			removeLinkFunc: func(_ context.Context, req *scrapperv1.RemoveLinkRequest, _ ...grpc.CallOption) (*scrapperv1.RemoveLinkResponse, error) {
				if req.GetChatId() != 1 {
					t.Fatalf("unexpected chat id: %d", req.GetChatId())
				}
				if req.GetBody().GetLink() != "https://github.com/user/repo" {
					t.Fatalf("unexpected link: %s", req.GetBody().GetLink())
				}
				return &scrapperv1.RemoveLinkResponse{
					Link: &scrapperv1.LinkResponse{
						Id:  11,
						Url: req.GetBody().GetLink(),
					},
				}, nil
			},
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

func listLinksSuccessCase(t *testing.T) struct {
	name   string
	call   func(context.Context, *GRPCClient) (any, error)
	assert func(*testing.T, any, error)
	client scrapperGRPCClient
} {
	t.Helper()
	return struct {
		name   string
		call   func(context.Context, *GRPCClient) (any, error)
		assert func(*testing.T, any, error)
		client scrapperGRPCClient
	}{
		name: "list links success",
		client: &fakeScrapperClient{
			listLinksFunc: func(_ context.Context, req *scrapperv1.ListLinksRequest, _ ...grpc.CallOption) (*scrapperv1.ListLinksResponse, error) {
				if req.GetChatId() != 1 {
					t.Fatalf("unexpected chat id: %d", req.GetChatId())
				}
				return &scrapperv1.ListLinksResponse{
					Links: []*scrapperv1.LinkResponse{{Id: 42, Url: "https://github.com/user/repo"}},
					Size:  1,
				}, nil
			},
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
		{name: "close success", closer: &fakeCloser{}, wantErr: false},
		{name: "close wraps error", closer: &fakeCloser{closeErr: errors.New("close failed")}, wantErr: true},
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

func TestFromProtoConversions(t *testing.T) {
	t.Parallel()

	t.Run("from proto link handles nil", func(t *testing.T) {
		t.Parallel()
		got := fromProtoLink(nil)
		if got.ID != 0 || got.URL != "" || len(got.Tags) != 0 || len(got.Filters) != 0 {
			t.Fatalf("unexpected non-zero value: %+v", got)
		}
	})

	t.Run("from proto list maps values", func(t *testing.T) {
		t.Parallel()
		got := fromProtoListLinks(&scrapperv1.ListLinksResponse{
			Links: []*scrapperv1.LinkResponse{
				{Id: 5, Url: "https://example.com", Tags: []string{"tag"}, Filters: []string{"f"}},
			},
			Size: 1,
		})
		if got.Size != 1 || len(got.Links) != 1 || got.Links[0].ID != 5 || got.Links[0].URL != "https://example.com" {
			t.Fatalf("unexpected conversion result: %+v", got)
		}
	})
}
