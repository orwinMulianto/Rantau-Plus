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
	client ChatClient
}

func NewService(client ChatClient) *Service {
	return &Service{client: client}
}

func (s *Service) chat(ctx context.Context, input Request) (Answer, error) {
	input.Message = strings.TrimSpace(input.Message)
	if input.Message == "" || utf8.RuneCountInString(input.Message) > 1000 {
		return Answer{}, errInvalidMessage
	}
	return s.client.Chat(ctx, input)
}
