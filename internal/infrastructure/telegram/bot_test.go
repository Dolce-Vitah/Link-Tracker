package telegram

import (
	"testing"

	command "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/commands"
	commandmock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/commands/mock"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	testifymock "github.com/stretchr/testify/mock"
)

func TestBot_HandleUpdate(t *testing.T) {
	bot := &Bot{
		dispatcher: command.NewDispatcher(),
	}
	bot.RegisterCommand(&command.StartCommand{}, &command.StartHandler{})
	bot.RegisterCommand(&command.HelpCommand{}, &command.HelpHandler{})

	tests := []struct {
		name           string
		command        string
		expectedAnswer string
	}{
		{
			name:           "Positive scenario: /start",
			command:        "start",
			expectedAnswer: "Добро пожаловать! Используйте /help, чтобы узнать о доступных командах.",
		},
		{
			name:           "Positive scenario: /help",
			command:        "help",
			expectedAnswer: "Доступные команды:\n/start - Начало работы с ботом\n/help - Показать этот список команд",
		},
		{
			name:           "Negative scenario: unknown command",
			command:        "abracadabra",
			expectedAnswer: "Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд.",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				mockSender = commandmock.NewSender(t)
				update     = &tgbotapi.Update{
					Message: &tgbotapi.Message{
						Text: "/" + tc.command,
						Chat: &tgbotapi.Chat{ID: 12345},
						From: &tgbotapi.User{
							ID:       98765,
							UserName: "test_user",
						},
						Entities: []tgbotapi.MessageEntity{
							{Type: "bot_command", Offset: 0, Length: len("/" + tc.command)},
						},
					},
				}
			)
			mockSender.On("Send", testifymock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil).Once()

			bot.handleUpdate(update, mockSender)

			mockSender.AssertExpectations(t)

			sendCall := mockSender.Calls[0]
			actualAnswer := sendCall.Arguments.Get(0).(tgbotapi.MessageConfig).Text
			if actualAnswer != tc.expectedAnswer {
				t.Errorf("\nОжидалось: %s\nПолучено:  %s", tc.expectedAnswer, actualAnswer)
			}
		})
	}
}
