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

// func (c *Client) Run() error {
// 	// Run forever... eventually have error handling, probably
// 	c.wg.Wait()
// 	return nil
// }

// func (c *Client) Stop() {
// 	// This disconnects from slack and closes the rtm channels
// 	err := c.rtm.Disconnect()
// 	if err != nil {
// 		// Non fatal error
// 		fmt.Printf("Failed to disconnect from RTM: %s\n", err.Error())
// 	}

// 	// Only close channels that have been created
// 	if c.outgoing != nil {
// 		fmt.Printf("Closing outgoing channel...\n")
// 		close(c.outgoing)
// 	}
// 	c.wg.Wait()
// }

// func (c *Client) CreateChannel(name string) (*slk.Channel, error) {
// 	channel, err := c.API.CreateConversation(name, false)
// 	if err != nil {
// 		return nil, err
// 	} else {
// 		return channel, nil
// 	}
// }

// func (c *Client) Hear(pattern string, fn func(*Message, [][]string)) {
// 	h := &MessageHandler{pattern: regexp.MustCompile(pattern), fn: fn}
// 	c.mHandler.Lock()
// 	c.handlers = append(c.handlers, h)
// 	c.mHandler.Unlock()
// }

// func (c *Client) Respond(pattern string, fn func(*Message, [][]string)) {
// 	c.Hear(fmt.Sprintf("<@%s|%s>:?\\W*%s", c.Username, c.userId, pattern), fn)
// }

// func (c *Client) Send(msg, channelId string) {
// 	c.mCnt.Lock()
// 	id := c.cnt
// 	c.cnt += 1
// 	c.mCnt.Unlock()

// 	c.outgoing <- slk.OutgoingMessage{
// 		ID:      id,
// 		Channel: channelId,
// 		Text:    msg,
// 		Type:    "message",
// 	}
// }

// func (c *Client) Pin(msg, channelId string) {
// 	c.outgoing <- slk.OutgoingMessage{
// 		Channel: channelId,
// 		Text:    msg,
// 		Type:    "pin",
// 	}
// }

// func (c *Client) GetPin(pattern *regexp.Regexp, channelID string) (string, error) {
// 	items, _, err := c.API.ListPins(channelID)
// 	if err != nil {
// 		return "", err
// 	}
// 	for _, m := range items {
// 		fmt.Println(m.Message)
// 		if pattern.Match([]byte(m.Message.Text)) {
// 			return m.Message.Text, nil
// 		}
// 	}
// 	return "", nil
// }

// func (c *Client) pinSlackMessage(channelId, msg string) error {
// 	// The channel is brand new, so there shouldn't be more than 100 messages in
// 	// it, which is the default count returned
// 	history, err := c.API.GetConversationHistory(&slk.GetConversationHistoryParameters{ChannelID: channelId})
// 	if err != nil {
// 		return fmt.Errorf("Failed to lookup channel history for %s: %s", channelId, err)
// 	}
// 	for _, post := range history.Messages {
// 		if post.Text == msg {
// 			// There should only be one message matching this text
// 			err := c.API.AddPin(channelId, slk.ItemRef{
// 				Channel:   post.Channel,
// 				Timestamp: post.Timestamp,
// 			})
// 			if err != nil {
// 				return fmt.Errorf("Failed to pin message: %s", err)
// 			}
// 			log.Printf("Pinned message `%s` in channel: %s\n", msg, channelId)
// 			return nil
// 		}
// 	}
// 	return fmt.Errorf("Could not find message with text `%s` in channel: %s", msg, channelId)
// }

// func (c *Client) start() {
// 	c.outgoing = make(chan slk.OutgoingMessage)

// 	// For every message posting, Flarebot needs certain parameters
// 	// set, so that it converses like another Slack user.
// 	messageParameters := slk.NewPostMessageParameters()
// 	messageParameters.LinkNames = 1
// 	messageParameters.AsUser = true
// 	optMessageParameters := slk.MsgOptionPostMessageParameters(messageParameters)

// 	c.wg.Add(1)
// 	go func(ws *slk.RTM, chSender chan slk.OutgoingMessage) error {
// 		for msg := range chSender {
// 			switch msg.Type {
// 			case "message":
// 				_, _, err := ws.PostMessage(msg.Channel, slk.MsgOptionText(msg.Text, false), optMessageParameters)
// 				if err != nil {
// 					return fmt.Errorf("Failed to post message. %s\n", err.Error())
// 				}
// 			case "pin":
// 				c.pinSlackMessage(msg.Channel, msg.Text)
// 			default:
// 				return fmt.Errorf("Unknown outgoing message type: %s", msg.Type)
// 			}
// 		}
// 		c.wg.Done()
// 		fmt.Printf("Outgoing goroutine ended\n")
// 		return nil
// 	}(c.rtm, c.outgoing)

// 	c.wg.Add(1)
// 	go func(rtm *slk.RTM) {
// 		// Not sure why, but MannageConnection() needs to be run as a goroutine
// 		// Otherwise it blocks forever
// 		go rtm.ManageConnection()
// 		for msg := range rtm.IncomingEvents {
// 			switch msg.Data.(type) {
// 			case *slk.HelloEvent:
// 				fmt.Println("Hello!")
// 			case *slk.MessageEvent:
// 				c.handleMessage(msg.Data.(*slk.MessageEvent))
// 			case *slk.RTMError:
// 				error := msg.Data.(*slk.RTMError)
// 				fmt.Printf("RTM error: %d - %s\n", error.Code, error.Msg)
// 			case *slk.UserTypingEvent, *slk.PresenceChangeEvent, slk.LatencyReport:
// 				// Do nothing
// 				continue
// 			case *slk.ConnectionErrorEvent:
// 				error := msg.Data.(*slk.ConnectionErrorEvent)
// 				fmt.Printf("Connection error: %v\n", error)
// 			case *slk.UnmarshallingErrorEvent:
// 				error := msg.Data.(*slk.UnmarshallingErrorEvent)
// 				fmt.Printf("Unmarshal event error: %v\n", error)
// 			case *slk.IncomingEventError:
// 				error := msg.Data.(*slk.IncomingEventError)
// 				fmt.Printf("Incoming event error: %v\n", error)
// 			case *slk.ChannelCreatedEvent:
// 				data := msg.Data.(*slk.ChannelCreatedEvent)
// 				fmt.Printf("Created channel: %s (%s)\n", data.Channel.Name, data.Channel.ID)
// 			case *slk.ChannelJoinedEvent:
// 				data := msg.Data.(*slk.ChannelJoinedEvent)
// 				fmt.Printf("Joined channel: %s (%s)\n", data.Channel.Name, data.Channel.ID)
// 			case *slk.ConnectedEvent:
// 				fmt.Printf("Connected to slack!\n")
// 			case *slk.ConnectingEvent:
// 				// Ignore the Connecting events
// 			case *slk.ChannelArchiveEvent:
// 				// Ignore the Channel Archive events
// 			case *slk.ChannelUnarchiveEvent:
// 				// Ignore the Channel Unarchive events
// 			case *slk.LatencyReport:
// 				// Ignore the Latency reports
// 			case *slk.ReactionAddedEvent:
// 				// Ignore the Reaction added events
// 			case *slk.ReactionRemovedEvent:
// 				// Ignore the Reaction removed events
// 			case *slk.EmojiChangedEvent:
// 				// Ignore the Emoji Changed events
// 			case *slk.FileCreatedEvent:
// 				// Ignore the File Created events
// 			case *slk.FileChangeEvent:
// 				// Ignore the File Change events
// 			case *slk.FilePublicEvent:
// 				// Ignore the file Public events
// 			case *slk.FileUnsharedEvent:
// 				// Ignore the File Unshared events
// 			case *slk.FileSharedEvent:
// 				// Ignore the File Shared events
// 			case *slk.FileDeletedEvent:
// 				// Ignore the File Deleted events
// 			case *slk.UserChangeEvent:
// 				// Ignore the User Change events
// 			case *slk.DNDUpdatedEvent:
// 				// Ignore the DNDUpdated events
// 			case *slk.ChannelHistoryChangedEvent:
// 				// Ignore the Channel History Changed events
// 			case *slk.BotAddedEvent:
// 				// Ignore the Bot added events
// 			case *slk.BotChangedEvent:
// 				// Ignore the Bot changed events
// 			case *slk.CommandsChangedEvent:
// 				// Ignore the commands changed events
// 			case *slk.TeamJoinEvent:
// 				// Ignore the team join events
// 			case *slk.ReconnectUrlEvent:
// 				// Ignore the reconnect URLS
// 			case *slk.PinAddedEvent:
// 				// Ignore the pin added events
// 			case *slk.PinRemovedEvent:
// 				// Ignore the pin removed events
// 			case *slk.ChannelMarkedEvent:
// 				// Ignore the channel marked events.
// 			case *slk.PrefChangeEvent:
// 				// Ignore the preference changed events
// 			case *slk.MemberJoinedChannelEvent, *slk.MemberLeftChannelEvent:
// 				// Ignore the join/leave channel events.
// 			case *slk.DisconnectedEvent:
// 				// If the disconnect was intentional, exit the goroutine
// 				data := msg.Data.(*slk.DisconnectedEvent)
// 				if data.Intentional {
// 					fmt.Printf("Disconnected from slack (intentionally)\n")
// 					fmt.Printf("Incoming goroutine ended\n")
// 					c.wg.Done()
// 					return
// 				}
// 			default:
// 				fmt.Printf("Unexpected: %#v\n", msg.Data)
// 			}
// 		}
// 		c.wg.Done()
// 		fmt.Printf("Incoming goroutine ended\n")
// 	}(c.rtm)
// }

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

	// Register all callbacks
	handlers := []*MessageHandler{}
	regexPattern := "<@%s|%s>:?\\W*%s"
	handlers = append(handlers, &MessageHandler{
		pattern: regexp.MustCompile(fmt.Sprintf(regexPattern, slackClient.Username, slackClient.UserID, fireFlareCommand.regexp)),
		fn:      slackClient.fireAFlareHandler,
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

// func NewClient(token, domain, username string, historyCallback func(message *Message) error) (*Client, error) {
// 	api := slk.New(token)

// 	users, err := api.GetUsers()
// 	if err != nil {
// 		return nil, fmt.Errorf("Failed to get users with error: %s", err)
// 	}
// 	var userId string
// 	for _, user := range users {
// 		if user.Name == username {
// 			userId = user.ID
// 			break
// 		}
// 	}

// 	rtm := api.NewRTM()
// 	client := &Client{API: api, rtm: rtm, Username: username, userId: userId, recordHistoryCallback: historyCallback}
// 	client.start()
// 	return client, nil
// }
