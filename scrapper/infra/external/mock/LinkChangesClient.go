package mock

import (
	"context"
	"time"

	mock "github.com/stretchr/testify/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/domain/linkchange"
)

func NewMockLinkChangesClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockLinkChangesClient {
	m := &MockLinkChangesClient{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

type MockLinkChangesClient struct {
	mock.Mock
}

func (_m *MockLinkChangesClient) FetchChangesSince(ctx context.Context, rawURL string, since time.Time) ([]linkchange.Change, time.Time, error) {
	ret := _m.Called(ctx, rawURL, since)
	var ch []linkchange.Change
	if r0 := ret.Get(0); r0 != nil {
		ch = r0.([]linkchange.Change)
	}
	var wm time.Time
	if r1 := ret.Get(1); r1 != nil {
		wm = r1.(time.Time)
	}
	return ch, wm, ret.Error(2)
}
