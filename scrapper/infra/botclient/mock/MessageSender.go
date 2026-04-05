package mock

import (
	"context"

	mock "github.com/stretchr/testify/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
)

func NewMockMessageSender(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockMessageSender {
	m := &MockMessageSender{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

type MockMessageSender struct {
	mock.Mock
}

func (_m *MockMessageSender) SendLinkUpdate(ctx context.Context, update trackerapi.LinkUpdate) error {
	ret := _m.Called(ctx, update)
	return ret.Error(0)
}

func (_m *MockMessageSender) SendProcessingFailureReport(ctx context.Context, report trackerapi.ProcessingFailureReport) error {
	ret := _m.Called(ctx, report)
	return ret.Error(0)
}
