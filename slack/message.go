package slack

import (
	"regexp"

	slk "github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

type Message struct {
	AuthorId   string
	AuthorName string
	Timestamp  string
	Text       string
	Channel    string
	api        *slk.Client
	sender     func(string, string)
}

func (m *Message) Author() (string, error) {
	user, err := m.AuthorUser()
	if err != nil {
		return "", err
	}
	return user.Name, nil
}

func (m *Message) AuthorUser() (*slk.User, error) {
	user, err := m.api.GetUserInfo(m.AuthorId)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func messageEventToMessage(evt *slackevents.MessageEvent, api *slk.Client) *Message {
	return &Message{
		AuthorId:   evt.BotID,
		AuthorName: evt.User,
		Timestamp:  evt.EventTimeStamp,
		Text:       evt.Text,
		Channel:    evt.Channel,
		api:        api,
	}
}

type MessageHandler struct {
	pattern *regexp.Regexp
	fn      func(*Message, [][]string)
}

func (h *MessageHandler) Match(msg *Message) bool {
	return h.pattern.Match([]byte(msg.Text))
}

func (h *MessageHandler) Handle(msg *Message) {
	h.fn(msg, h.pattern.FindAllStringSubmatch(msg.Text, -1))
}
