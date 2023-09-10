package slack

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/modern-pet/flarebot/googledocs"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

//
// COMMANDS
//

type command struct {
	regexp      string
	example     string
	description string
}

var fireFlareCommand = &command{
	regexp:      "[fF]ire (?:a )?(?:retroactive )?(?:.+emptive )?[fF]lare [pP]([012]) *(.*)",
	example:     "fire a flare p2 there is still no hottub on the roof",
	description: "Fire a new Flare with the given priority and description",
}

var testCommand = &command{
	regexp:      "test *(.*)",
	example:     "",
	description: "",
}

var takingLeadCommand = &command{
	regexp:      "[iI]('?m?| am?) (the )?incident lead",
	example:     "I am incident lead",
	description: "Declare yourself incident lead.",
}

var flareMitigatedCommand = &command{
	regexp:      "([Ff]lare )?(is )?mitigated",
	example:     "flare mitigated",
	description: "Mark the Flare mitigated.",
}

// not a flare
var notAFlareCommand = &command{
	regexp:      "([Ff]lare )?(is )?not a [Ff]lare",
	example:     "not a flare",
	description: "Mark the Flare not-a-flare.",
}

// help command
var helpCommand = &command{
	regexp:      "[Hh]elp *$",
	example:     "help",
	description: "display the list of commands available in this channel.",
}

// help all command
var helpAllCommand = &command{
	regexp:      "[Hh]elp [Aa]ll",
	example:     "help all",
	description: "display the list of all commands and the channels where they're available.",
}

var mainChannelCommands = []*command{helpCommand, helpAllCommand, fireFlareCommand}
var flareChannelCommands = []*command{helpCommand, takingLeadCommand, flareMitigatedCommand, notAFlareCommand}
var otherChannelCommands = []*command{helpAllCommand}

type SlackClient struct {
	Client                  *socketmode.Client
	Username                string
	UserID                  string
	ExpectedChannel         string
	GoogleDocsServer        *googledocs.GoogleDocsServer
	GoogleDomain            string
	GoogleFlareDocID        string
	GoogleSlackHistoryDocID string
	handlers                []*MessageHandler
}

func NewSlackClient(username string, expectedChannel string, googleDocsServer *googledocs.GoogleDocsServer, googleDomain string, googleFlareDocID string, googleSlackHistoryDocID string) (*SlackClient, error) {
	appToken := os.Getenv("SLACK_FLAREBOT_APP_ACCESS_TOKEN")
	if appToken == "" {
		return nil, errors.New("SLACK_FLAREBOT_APP_ACCESS_TOKEN must be set")
	}

	if !strings.HasPrefix(appToken, "xapp-") {
		return nil, errors.New("SLACK_FLAREBOT_APP_ACCESS_TOKEN must have the prefix \"xapp-\".")
	}

	botToken := os.Getenv("SLACK_FLAREBOT_BOT_ACCESS_TOKEN")
	if botToken == "" {
		return nil, errors.New("SLACK_FLAREBOT_BOT_ACCESS_TOKEN must be set.")
	}

	if !strings.HasPrefix(botToken, "xoxb-") {
		return nil, errors.New("SLACK_FLAREBOT_BOT_ACCESS_TOKEN must have the prefix \"xoxb-\".")
	}

	api := slack.New(
		botToken,
		slack.OptionDebug(true),
		slack.OptionAppLevelToken(appToken),
		slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
	)

	client := socketmode.New(
		api,
		socketmode.OptionDebug(true),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	users, err := api.GetUsers()
	if err != nil {
		return nil, fmt.Errorf("Failed to get users with error: %s", err)
	}
	var userId string
	for _, user := range users {
		if user.Name == username {
			userId = user.ID
			break
		}
	}

	slackClient := &SlackClient{
		Client:                  client,
		Username:                username,
		UserID:                  userId,
		ExpectedChannel:         expectedChannel,
		GoogleDocsServer:        googleDocsServer,
		GoogleDomain:            googleDomain,
		GoogleFlareDocID:        googleFlareDocID,
		GoogleSlackHistoryDocID: googleSlackHistoryDocID,
	}

	// Register all handlers
	handlers := []*MessageHandler{}
	regexPattern := "<@%s|%s>:?\\W*%s"
	handlers = append(handlers, &MessageHandler{
		pattern: regexp.MustCompile(fmt.Sprintf(regexPattern, slackClient.Username, slackClient.UserID, fireFlareCommand.regexp)),
		fn:      slackClient.fireAFlareHandler,
	})
	handlers = append(handlers, &MessageHandler{
		pattern: regexp.MustCompile(fmt.Sprintf(regexPattern, slackClient.Username, slackClient.UserID, takingLeadCommand.regexp)),
		fn:      slackClient.takingLeadHandler,
	})
	handlers = append(handlers, &MessageHandler{
		pattern: regexp.MustCompile(fmt.Sprintf(regexPattern, slackClient.Username, slackClient.UserID, flareMitigatedCommand.regexp)),
		fn:      slackClient.mitigateFlareHandler,
	})
	handlers = append(handlers, &MessageHandler{
		pattern: regexp.MustCompile(fmt.Sprintf(regexPattern, slackClient.Username, slackClient.UserID, notAFlareCommand.regexp)),
		fn:      slackClient.notAFlareHandler,
	})
	handlers = append(handlers, &MessageHandler{
		pattern: regexp.MustCompile(fmt.Sprintf(regexPattern, slackClient.Username, slackClient.UserID, helpCommand.regexp)),
		fn:      slackClient.helpHandler,
	})
	handlers = append(handlers, &MessageHandler{
		pattern: regexp.MustCompile(fmt.Sprintf(regexPattern, slackClient.Username, slackClient.UserID, helpAllCommand.regexp)),
		fn:      slackClient.helpAllHandler,
	})
	handlers = append(handlers, &MessageHandler{
		pattern: regexp.MustCompile(fmt.Sprintf(regexPattern, slackClient.Username, slackClient.UserID, ".*")),
		fn:      slackClient.otherHandlers,
	})

	slackClient.handlers = handlers

	go func() {
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeConnecting:
				fmt.Println("Connecting to Slack with Socket Mode...")
			case socketmode.EventTypeConnectionError:
				fmt.Println("Connection failed. Retrying later...")
			case socketmode.EventTypeConnected:
				fmt.Println("Connected to Slack with Socket Mode.")
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					fmt.Printf("Ignored %+v\n", evt)
					continue
				}

				fmt.Printf("Event received: %+v\n", eventsAPIEvent)

				client.Ack(*evt.Request)

				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					innerEvent := eventsAPIEvent.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.MessageEvent:
						slackClient.handleMessage(ev)
					}
				default:
					client.Debugf("unsupported Events API event received")
				}
			default:
				fmt.Fprintf(os.Stderr, "Unexpected event type received: %s\n", evt.Type)
			}
		}
	}()

	return slackClient, nil
}

func (c *SlackClient) handleMessage(evt *slackevents.MessageEvent) {
	m := messageEventToMessage(evt, &c.Client.Client)

	var theMatch *MessageHandler

	// If the message is from us, don't do anything
	author, _ := m.Author()
	if author == c.Username {
		fmt.Println("Message is from us, skipping -------------------------")
		return
	}

	for _, h := range c.handlers {
		if h.Match(m) {
			theMatch = h
			break
		}
	}

	if theMatch != nil {
		theMatch.Handle(m)
	}

	c.recordSlackHistory(m)
}
