package assistant

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrUnknownSkill   = errors.New("skill belum tersedia")
	errInvalidMessage = errors.New("isi pesan harus 1-1000 karakter")
)

type ChatClient interface {
	Chat(context.Context, Request) (Answer, error)
}

type Service struct {
	client  ChatClient
	places  FoodFinder
	history conversations
}

func NewCompanion(client ChatClient, places FoodFinder) *Service {
	return &Service{client: client, places: places}
}

func NewService(client ChatClient) *Service {
	return &Service{client: client}
}

func (s *Service) chat(ctx context.Context, input Request) (Answer, error) {
	message, err := validateMessage(input.Message)
	if err != nil {
		return Answer{}, err
	}
	input.Message = message
	return s.client.Chat(ctx, input)
}

func validateMessage(message string) (string, error) {
	message = strings.TrimSpace(message)
	if message == "" || utf8.RuneCountInString(message) > 1000 {
		return "", errInvalidMessage
	}
	return message, nil
}
