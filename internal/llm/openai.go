package llm

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type Message struct {
	Role    string
	Content string
}

type Client interface {
	Chat(ctx context.Context, sys string, msgs []Message) (string, error)
}

type openAI struct {
	c     *openai.Client
	model string
}

// ⚙️ Конфигурация: ключ и модель по умолчанию
const (
	defaultKey   = "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"
	defaultModel = "z-ai/glm-4.5-air:free"
	defaultBase  = "https://openrouter.ai/api/v1"
)

func NewOpenAI(model string) (Client, error) {
	cfg := openai.DefaultConfig(defaultKey)
	cfg.BaseURL = defaultBase

	client := openai.NewClientWithConfig(cfg)
	m := model
	if m == "" {
		m = defaultModel
	}
	return &openAI{c: client, model: m}, nil
}

func (o *openAI) Chat(ctx context.Context, sys string, msgs []Message) (string, error) {
	om := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: sys},
	}
	for _, m := range msgs {
		om = append(om, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	req := openai.ChatCompletionRequest{
		Model:       o.model,
		Messages:    om,
		Temperature: 0.2,
	}

	resp, err := o.c.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices")
	}
	return resp.Choices[0].Message.Content, nil
}
