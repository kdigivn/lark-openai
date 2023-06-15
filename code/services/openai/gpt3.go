package openai

import (
	"errors"
	"strings"

	"github.com/pandodao/tokenizer-go"
)

type AIMode float64

const (
	Fresh      AIMode = 0.1
	Warmth     AIMode = 0.4
	Balance    AIMode = 0.7
	Creativity AIMode = 1.0
)

var AIModeMap = map[string]AIMode{
	"Fresh": Fresh,
	"Warmth": Warmth,
	"Balance": Balance,
	"Creativity": Creativity,
}

var AIModeStrs = []string{
	"Fresh",
	"Warmth",
	"Balance",
	"Creativity",
}

const (
	maxTokens = 2000
	engine    = "gpt-3.5-turbo"
)

type Messages struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatGPTResponseBody 请求体
type ChatGPTResponseBody struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int                    `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatGPTChoiceItem    `json:"choices"`
	Usage   map[string]interface{} `json:"usage"`
}

type ChatGPTChoiceItem struct {
	Message      Messages `json:"message"`
	Index        int      `json:"index"`
	FinishReason string   `json:"finish_reason"`
}

// ChatGPTRequestBody 响应体
type ChatGPTRequestBody struct {
	Model            string     `json:"model"`
	Messages         []Messages `json:"messages"`
	MaxTokens        int        `json:"max_tokens"`
	Temperature      AIMode     `json:"temperature"`
	TopP             int        `json:"top_p"`
	FrequencyPenalty int        `json:"frequency_penalty"`
	PresencePenalty  int        `json:"presence_penalty"`
}

func (msg *Messages) CalculateTokenLength() int {
	text := strings.TrimSpace(msg.Content)
	return tokenizer.MustCalToken(text)
}

func (gpt *ChatGPT) Completions(msg []Messages, aiMode AIMode) (resp Messages,
	err error) {
	requestBody := ChatGPTRequestBody{
		Model:            engine,
		Messages:         msg,
		MaxTokens:        maxTokens,
		Temperature:      aiMode,
		TopP:             1,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
	}
	gptResponseBody := &ChatGPTResponseBody{}
	url := gpt.FullUrl("chat/completions")
	//fmt.Println(url)
	if url == "" {
		return resp, errors.New("Không thể lấy địa chỉ yêu cầu openai")
	}
	err = gpt.sendRequestWithBodyType(url, "POST", jsonBody, requestBody, gptResponseBody)
	if err == nil && len(gptResponseBody.Choices) > 0 {
		resp = gptResponseBody.Choices[0].Message
	} else {
		resp = Messages{}
		err = errors.New("OpenAI: Yêu cầu không thành công")
	}
	return resp, err
}
