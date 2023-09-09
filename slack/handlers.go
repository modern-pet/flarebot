package slack

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/modern-pet/flarebot/helpers"
	"github.com/slack-go/slack"
)

var slackHistoryDocCache = map[string]string{}

func (c *SlackClient) fireAFlareHandler(msg *Message, params [][]string) {
	// wrong channel?
	if msg.Channel != c.ExpectedChannel {
		return
	}

	log.Printf("starting flare process. I was told %s", msg.Text)

	c.Client.SetUserAsActive()

	// retroactive?
	isRetroactive := strings.Contains(msg.Text, "retroactive")
	// preemptive?
	isPreemptive := strings.Contains(msg.Text, "emptive") // this could be pre-emptive, or preemptive

	if isRetroactive {
		c.Client.PostMessage(msg.Channel, slack.MsgOptionText("OK, let me quietly set up the Flare documents. Nobody freak out, this is retroactive.", false))
	} else if isPreemptive {
		c.Client.PostMessage(msg.Channel, slack.MsgOptionText("OK, let me quietly set up the Flare documents. Nobody freak out, this is preemptive.", false))
	} else {
		c.Client.PostMessage(msg.Channel, slack.MsgOptionText("OK, let me get my flaregun", false))
	}

	// for now matches are indexed
	topic := params[0][2]

	flareDocTitle := fmt.Sprintf("%s: %s", "Flare", topic)

	if isRetroactive {
		flareDocTitle = fmt.Sprintf("%s - Retroactive", flareDocTitle)
	}

	log.Printf("Attempting to create flare doc")
	flareDoc, flareDocErr := c.GoogleDocsServer.CreateFromTemplate(flareDocTitle, c.GoogleFlareDocID, map[string]string{})

	if flareDocErr != nil {
		c.Client.PostMessage(msg.Channel, slack.MsgOptionText("I'm having trouble connecting to google docs right now, so I can't make a flare doc for tracking. I'll try my best to recover.", false))
		log.Printf("No google flare doc created: %s", flareDocErr)
	} else {
		log.Printf("Flare doc created")
	}

	log.Printf("Attempting to create history doc")
	slackHistoryDocTitle := fmt.Sprintf("%s: %s (Slack History)", "Flare", topic)
	slackHistoryDoc, historyDocErr := c.GoogleDocsServer.CreateFromTemplate(slackHistoryDocTitle, c.GoogleSlackHistoryDocID, map[string]string{})

	if historyDocErr != nil {
		log.Printf("No google slack history doc created: %s", historyDocErr)
	} else {
		log.Printf("Google slack history doc created")
	}

	if flareDocErr == nil {
		// update the google doc with some basic information
		html, err := c.GoogleDocsServer.GetDocContent(flareDoc, "text/html")
		if err != nil {
			log.Printf("unexpected errror getting content from the flare doc: %s", err)
		} else {
			date, err := helpers.GetJakartaDateAndTime()
			if err != nil {
				log.Printf("Failed to get jakarta time now: %s", err)
			}
			html = strings.Replace(html, "[START-DATE]", date, 1)
			html = strings.Replace(html, "[SUMMARY]", topic, 1)
			html = strings.Replace(html, "[HISTORY-DOC]",
				fmt.Sprintf(`<a href="%s">%s</a>`, slackHistoryDoc.File.AlternateLink, slackHistoryDocTitle), 1)

			c.GoogleDocsServer.UpdateDocContent(flareDoc, html)

			// update permissions
			if err = c.GoogleDocsServer.ShareDocWithDomain(flareDoc, c.GoogleDomain, "writer"); err != nil {
				// It's OK if we continue here, and don't error out
				log.Printf("Couldn't share google flare doc: %s", err)
			}
			if err = c.GoogleDocsServer.ShareDocWithDomain(slackHistoryDoc, c.GoogleDomain, "writer"); err != nil {
				// It's OK if we continue here, and don't error out
				log.Printf("Couldn't share google slack history doc: %s", err)
			}
		}
	}

	log.Printf("Attempting to create flare channel")
	// // set up the Flare room
	flareID := fmt.Sprintf("flare-%s", uuid.New().String())
	channel, channelErr := c.Client.CreateConversation(slack.CreateConversationParams{ChannelName: flareID, IsPrivate: false})
	if channelErr != nil {
		c.Client.PostMessage(msg.Channel, slack.MsgOptionText("Slack is giving me some trouble right now, so I couldn't create a channel for you. It could be that the channel already exists, but hopefully no one did that already. If you need to make a new channel to discuss, please don't use the next flare-number channel, that'll confuse me later on.", false))
		log.Printf("Couldn't create Flare channel: %s", channelErr)
	} else {
		log.Printf("Flare channel created")

		if isRetroactive {
			c.Client.PostMessage(channel.ID, slack.MsgOptionText("This is a RETROACTIVE Flare. All is well.", false))
		}

		c.Client.SetTopicOfConversation(channel.ID, topic)

		if flareDocErr == nil {
			c.Client.PostMessage(channel.ID, slack.MsgOptionText(fmt.Sprintf("Flare doc: %s", flareDoc.File.AlternateLink), false))
		}
		if historyDocErr == nil {
			slackHistoryDocCache[channel.ID] = slackHistoryDoc.File.Id
			c.Client.PostMessage(channel.ID, slack.MsgOptionText(fmt.Sprintf("Slack log: %s", slackHistoryDoc.File.Id), false))
		}
		c.Client.PostMessage(channel.ID, slack.MsgOptionText(fmt.Sprintf("Remember: Rollback, Scale or Restart!"), false))

		if flareDocErr == nil {
			c.Client.AddPin(channel.ID, slack.ItemRef{Comment: fmt.Sprintf("Flare doc: <%s>", flareDoc.File.AlternateLink)})
		}
		if historyDocErr == nil {
			c.Client.AddPin(channel.ID, slack.ItemRef{Comment: fmt.Sprintf("Slack log: %s", slackHistoryDoc.File.Id)})
		}
		c.Client.AddPin(channel.ID, slack.ItemRef{Comment: fmt.Sprintf("Remember: Rollback, Scale or Restart!")})

		// send room-specific help
		c.sendHelpMessage(channel.ID, false)

		// let people know that they can rename this channel
		c.Client.PostMessage(channel.ID, slack.MsgOptionText(fmt.Sprintf("NOTE: you can rename this channel as long as it starts with %s", channel.Name), false))

		// announce the specific Flare room in the overall Flares room
		target := "channel"

		if isRetroactive || isPreemptive {
			author, _ := msg.AuthorUser()
			target = author.Name
		}

		c.Client.PostMessage(msg.Channel, slack.MsgOptionText(fmt.Sprintf("<!%s>: Flare fired. Please visit <#%s> -- %s", target, channel.ID, topic), false))
	}
}

func (c *SlackClient) sendHelpMessage(channel string, inMainChannel bool) {
	availableCommands := mainChannelCommands

	c.Client.PostMessage(channel, slack.MsgOptionText("Available commands:", false))
	c.sendCommandsHelpMessage(channel, availableCommands)
}

func (c *SlackClient) sendCommandsHelpMessage(channel string, commands []*command) {
	for _, command := range commands {
		c.Client.PostMessage(channel, slack.MsgOptionText(fmt.Sprintf("\"@%s: %s\" - %s", c.Username, command.example, command.description), false))
	}
}

func (c *SlackClient) recordSlackHistory(message *Message) error {
	docID, ok := slackHistoryDocCache[message.Channel]
	if !ok {
		channel, err := c.Client.GetConversationInfo(&slack.GetConversationInfoInput{ChannelID: message.Channel})
		if err != nil {
			return err
		}

		docID = ""
		if regexp.MustCompile("^flare-").Match([]byte(channel.Name)) {
			// Get pinned link
			historyPin := regexp.MustCompile("^Slack log: (.*)")
			pins, _, err := c.Client.ListPins(message.Channel)
			if err != nil {
				// There might not be a pin in this channel, just ignore it.
				fmt.Printf("Unable to get Slack log pin for %s, skipping\n", channel.Name)
			} else {
				for _, pin := range pins {
					if len(historyPin.FindStringSubmatch(pin.Comment.Comment)) > 0 {
						docID = historyPin.FindStringSubmatch(pin.Comment.Comment)[1]
					}
				}

			}
		}

		// And write it back for caching purposes.
		slackHistoryDocCache[message.Channel] = docID
	}

	// If there's no doc, don't record the history. Not all channels need one.
	if docID == "" {
		return nil
	}

	var formattedTime = strings.Split(message.Timestamp, ".")[0]
	author, err := message.Author()
	if err != nil {
		author = message.AuthorId
	}

	data := []interface{}{
		message.Timestamp,
		formattedTime,
		author,
		message.Text,
	}

	doc, err := c.GoogleDocsServer.GetDoc(docID)
	if err != nil {
		fmt.Println("Unable to find slack history doc")
		return err
	}

	err = c.GoogleDocsServer.AppendSheetContent(doc, data)
	if err != nil {
		fmt.Printf("Unable to write slack history: %s", err)
		return err
	}

	return nil
}
