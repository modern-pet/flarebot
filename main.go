package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Clever/flarebot/googledocs"
	"github.com/Clever/flarebot/slack"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
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

// #flare-179-foo-bar --> #flare-179
var channelNameRegexp = regexp.MustCompile("^([^-]+-[^-]+)(?:-.+)")
var flareChannelNamePrefix *regexp.Regexp

// Save slack history doc IDs in a cache for more efficient lookups
var slackHistoryDocCache = map[string]string{}

func recordSlackHistory(client *slack.Client, googleDocsServer googledocs.GoogleDocsService, message *slack.Message) error {
	docID, ok := slackHistoryDocCache[message.Channel]
	if !ok {
		channel, err := client.API.GetConversationInfo(message.Channel, false)
		if err != nil {
			return err
		}

		docID = ""
		if flareChannelNamePrefix.Match([]byte(channel.Name)) {
			// Get pinned link
			historyPin := regexp.MustCompile("^Slack log: (.*)")
			pin, err := client.GetPin(historyPin, message.Channel)
			if err != nil {
				// There might not be a pin in this channel, just ignore it.
				fmt.Printf("Unable to get Slack log pin for %s, skipping\n", channel.Name)
			} else {
				docID = historyPin.FindStringSubmatch(pin)[1]
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
	unixTime, err := strconv.ParseInt(formattedTime, 10, 64)
	if err == nil {
		formattedTime = timeStringInTZ(time.Unix(unixTime, 0), "US/Pacific")
	}
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

	doc, err := googleDocsServer.GetDoc(docID)
	if err != nil {
		fmt.Println("Unable to find slack history doc")
		return err
	}

	err = googleDocsServer.AppendSheetContent(doc, data)
	if err != nil {
		fmt.Println("Unable to write slack history")
		return err
	}

	return nil
}

// currentTimeStringInTZ returns the current time in a TZ format as determined by Golang's "Location" type.
func currentTimeStringInTZ(tz string) string {
	return timeStringInTZ(time.Now(), tz)
}

// timeStringInTZ returns the time in a TZ format as determined by Golang's "Location" type.
func timeStringInTZ(t time.Time, tz string) string {
	tzLocation, _ := time.LoadLocation(tz)
	return t.In(tzLocation).Format(time.RFC3339)
}

func sendCommandsHelpMessage(client *slack.Client, channel string, commands []*command) {
	for _, c := range commands {
		client.Send(fmt.Sprintf("\"@%s: %s\" - %s", client.Username, c.example, c.description), channel)
	}
}

func sendHelpMessage(client *slack.Client, channel string, inMainChannel bool) {
	var availableCommands []*command = mainChannelCommands

	if len(availableCommands) == 0 {
		client.Send("no available commands in this channel.", channel)
	} else {
		client.Send("Available commands:", channel)
		sendCommandsHelpMessage(client, channel, availableCommands)
	}
}

func sendReminderMessage(client *slack.Client, channel string, message string, delay time.Duration) {
	time.Sleep(delay)
	client.Send(message, channel)
}

func main() {
	godotenv.Load()

	flareChannelNamePrefix = regexp.MustCompile("flare-")

	// Google Docs service
	googleDocsServer, err := googledocs.NewGoogleDocsServerWithServiceAccount(os.Getenv("GOOGLE_FLAREBOT_SERVICE_ACCOUNT_CONF"))
	googleDomain := os.Getenv("GOOGLE_DOMAIN")

	googleFlareDocID := os.Getenv("GOOGLE_TEMPLATE_DOC_ID")
	googleSlackHistoryDocID := os.Getenv("GOOGLE_SLACK_HISTORY_DOC_ID")

	// Link to flare resources
	resources_url := os.Getenv("FLARE_RESOURCES_URL")

	// Link to status page
	status_page_login_url := os.Getenv("STATUS_PAGE_LOGIN_URL")

	// Slack connection params
	var accessToken string
	if os.Getenv("SLACK_LEGACY_TOKEN") != "" {
		accessToken = os.Getenv("SLACK_FLAREBOT_ACCESS_TOKEN")
	} else {
		accessToken = os.Getenv("SLACK_FLAREBOT_ACCESS_TOKEN")
	}
	domain := os.Getenv("SLACK_DOMAIN")
	username := os.Getenv("SLACK_USERNAME")

	var client *slack.Client
	recordSlackHistoryCallback := func(message *slack.Message) error {
		return recordSlackHistory(client, googleDocsServer, message)
	}

	client, err = slack.NewClient(accessToken, domain, username, recordSlackHistoryCallback)
	if err != nil {
		panic(err)
	}

	expectedChannel := os.Getenv("SLACK_CHANNEL")

	client.Respond(testCommand.regexp, func(msg *slack.Message, params [][]string) {
		author, err := msg.AuthorUser()
		if err != nil {
			client.Send("Unable to determine author of Slack message", msg.Channel)
		}
		client.Send(fmt.Sprintf("I see you're using the test command. Excellent: %s", author.Profile.Email), msg.Channel)

		if len(params[0]) > 1 {
			client.Send(fmt.Sprintf("you told me: %s", params[0][1]), msg.Channel)
		}

		channel, err := client.API.GetConversationInfo(msg.Channel, false)
		if err != nil {
			client.Send("Unable to determine channel info", msg.Channel)
		}

		client.Send(fmt.Sprintf("this channel is %s", channel.Name), msg.Channel)

		// verify that we can open and parse the FLARE template
		flareDoc, err := googleDocsServer.GetDoc(googleFlareDocID)
		if err != nil {
			client.Send(fmt.Sprintf("Unable to find the Google Doc Flare Template. ID: %s", googleFlareDocID), msg.Channel)
			return
		}

		_, err = googleDocsServer.GetDocContent(flareDoc, "text/html")
		if err != nil {
			client.Send(fmt.Sprintf("Unable to get Google Doc Content for ID: %s", googleFlareDocID), msg.Channel)
			return
		}

		// And Slack History template
		slackHistoryDoc, err := googleDocsServer.GetDoc(googleSlackHistoryDocID)
		if err != nil {
			client.Send(fmt.Sprintf("Unable to find the Google Slack History Doc Template. ID: %s", googleSlackHistoryDocID), msg.Channel)
			return
		}

		_, err = googleDocsServer.GetSheetContent(slackHistoryDoc)
		if err != nil {
			client.Send(fmt.Sprintf("Unable to get Google Doc Content for ID: %s", googleSlackHistoryDocID), msg.Channel)
			return
		}
	})

	client.Respond(fireFlareCommand.regexp, func(msg *slack.Message, params [][]string) {
		// wrong channel?
		if msg.Channel != expectedChannel {
			return
		}

		log.Printf("starting flare process. I was told %s", msg.Text)

		client.API.SetUserAsActive()

		// retroactive?
		isRetroactive := strings.Contains(msg.Text, "retroactive")
		// preemptive?
		isPreemptive := strings.Contains(msg.Text, "emptive") // this could be pre-emptive, or preemptive

		if isRetroactive {
			client.Send("OK, let me quietly set up the Flare documents. Nobody freak out, this is retroactive.", msg.Channel)
		} else if isPreemptive {
			client.Send("OK, let me quietly set up the Flare documents. Nobody freak out, this is preemptive.", msg.Channel)
		} else {
			client.Send("OK, let me get my flaregun", msg.Channel)
		}

		// for now matches are indexed
		topic := params[0][2]

		flareDocTitle := fmt.Sprintf("%s: %s", "Flare", topic)

		if isRetroactive {
			flareDocTitle = fmt.Sprintf("%s - Retroactive", flareDocTitle)
		}

		log.Printf("Attempting to create flare doc")
		flareDoc, flareDocErr := googleDocsServer.CreateFromTemplate(flareDocTitle, googleFlareDocID, map[string]string{})

		if flareDocErr != nil {
			client.Send("I'm having trouble connecting to google docs right now, so I can't make a flare doc for tracking. I'll try my best to recover.", msg.Channel)
			log.Printf("No google flare doc created: %s", flareDocErr)
		} else {
			log.Printf("Flare doc created")
		}

		log.Printf("Attempting to create history doc")
		slackHistoryDocTitle := fmt.Sprintf("%s: %s (Slack History)", "Flare", topic)
		slackHistoryDoc, historyDocErr := googleDocsServer.CreateFromTemplate(slackHistoryDocTitle, googleSlackHistoryDocID, map[string]string{})

		if historyDocErr != nil {
			log.Printf("No google slack history doc created: %s", historyDocErr)
		} else {
			log.Printf("Google slack history doc created")
		}

		if flareDocErr == nil {
			// update the google doc with some basic information
			html, err := googleDocsServer.GetDocContent(flareDoc, "text/html")
			if err != nil {
				log.Printf("unexpected errror getting content from the flare doc: %s", err)
			} else {
				html = strings.Replace(html, "[START-DATE]", currentTimeStringInTZ("US/Pacific"), 1)
				html = strings.Replace(html, "[SUMMARY]", topic, 1)
				html = strings.Replace(html, "[HISTORY-DOC]",
					fmt.Sprintf(`<a href="%s">%s</a>`, slackHistoryDoc.File.AlternateLink, slackHistoryDocTitle), 1)

				googleDocsServer.UpdateDocContent(flareDoc, html)

				// update permissions
				if err = googleDocsServer.ShareDocWithDomain(flareDoc, googleDomain, "writer"); err != nil {
					// It's OK if we continue here, and don't error out
					log.Printf("Couldn't share google flare doc: %s", err)
				}
				if err = googleDocsServer.ShareDocWithDomain(slackHistoryDoc, googleDomain, "writer"); err != nil {
					// It's OK if we continue here, and don't error out
					log.Printf("Couldn't share google slack history doc: %s", err)
				}
			}
		}

		log.Printf("Attempting to create flare channel")
		// set up the Flare room
		flareID := strings.ToLower(uuid.New().String())
		channel, channelErr := client.CreateChannel(flareID)
		if channelErr != nil {
			client.Send("Slack is giving me some trouble right now, so I couldn't create a channel for you. It could be that the channel already exists, but hopefully no one did that already. If you need to make a new channel to discuss, please don't use the next flare-number channel, that'll confuse me later on.", msg.Channel)
			log.Printf("Couldn't create Flare channel: %s", channelErr)
		} else {
			log.Printf("Flare channel created")

			if isRetroactive {
				client.Send("This is a RETROACTIVE Flare. All is well.", channel.ID)
			}

			client.API.SetTopicOfConversation(channel.ID, topic)

			if flareDocErr == nil {
				client.Send(fmt.Sprintf("Flare doc: %s", flareDoc.File.AlternateLink), channel.ID)
			}
			if historyDocErr == nil {
				slackHistoryDocCache[channel.ID] = slackHistoryDoc.File.Id
				client.Send(fmt.Sprintf("Slack log: %s", slackHistoryDoc.File.Id), channel.ID)
			}
			client.Send(fmt.Sprintf("Flare resources: %s", resources_url), channel.ID)
			client.Send(fmt.Sprintf("Manage status page: %s", status_page_login_url), channel.ID)
			client.Send(fmt.Sprintf("Remember: Rollback, Scale or Restart!"), channel.ID)

			if flareDocErr == nil {
				client.Pin(fmt.Sprintf("Flare doc: <%s>", flareDoc.File.AlternateLink), channel.ID)
			}
			if historyDocErr == nil {
				client.Pin(fmt.Sprintf("Slack log: %s", slackHistoryDoc.File.Id), channel.ID)
			}
			client.Pin(fmt.Sprintf("Manage status page: <%s>", status_page_login_url), channel.ID)
			client.Pin(fmt.Sprintf("Remember: Rollback, Scale or Restart!"), channel.ID)

			// send room-specific help
			sendHelpMessage(client, channel.ID, false)

			// let people know that they can rename this channel
			client.Send(fmt.Sprintf("NOTE: you can rename this channel as long as it starts with %s", channel.Name), channel.ID)

			go sendReminderMessage(client, channel.ID, "Are the right people in the flare channel? Consider using the /page Slack command.", 3*time.Minute)
			go sendReminderMessage(client, channel.ID, "Have you tried rolling back, scaling or restarting? (consider SSO version too)", 5*time.Minute)

			// announce the specific Flare room in the overall Flares room
			target := "channel"

			if isRetroactive || isPreemptive {
				author, _ := msg.AuthorUser()
				target = author.Name
			}

			client.Send(fmt.Sprintf("@%s: Flare fired. Please visit #%s -- %s", target, flareID, topic), msg.Channel)
		}
	})

	client.Respond(takingLeadCommand.regexp, func(msg *slack.Message, params [][]string) {
		author, _ := msg.AuthorUser()

		client.Send("working on assigning incident lead....", msg.Channel)

		client.Send(fmt.Sprintf("Oh Captain My Captain! @%s is now incident lead. Please confirm all actions with them.", author.Name), msg.Channel)
	})

	client.Respond(flareMitigatedCommand.regexp, func(msg *slack.Message, params [][]string) {
		// notify the main flares channel
		client.Send("Flare has been mitigated", expectedChannel)
	})

	client.Respond(notAFlareCommand.regexp, func(msg *slack.Message, params [][]string) {
		// notify the main flares channel
		client.Send("turns out this is not a Flare", expectedChannel)
	})

	client.Respond(helpCommand.regexp, func(msg *slack.Message, params [][]string) {
		sendHelpMessage(client, msg.Channel, (msg.Channel == expectedChannel))
	})

	client.Respond(helpAllCommand.regexp, func(msg *slack.Message, param [][]string) {
		client.Send("Commands Available in the #flares channel:", msg.Channel)
		sendCommandsHelpMessage(client, msg.Channel, mainChannelCommands)
		client.Send("Commands Available in a single Flare channel:", msg.Channel)
		sendCommandsHelpMessage(client, msg.Channel, flareChannelCommands)
		client.Send("Commands Available in other channels:", msg.Channel)
		sendCommandsHelpMessage(client, msg.Channel, otherChannelCommands)
	})

	// fallback response saying "I don't understand"
	client.Respond(".*", func(msg *slack.Message, params [][]string) {
		// should be taking commands here, and didn't understand
		client.Send(`I'm sorry, I didn't understand that command.
			To fire a flare: @flarebot fire a flare <p0|p1|p2> [pre-emptive|retroactive] <problem>
			For other commands: @flarebot help [all]
		`, msg.Channel)
	})

	panic(client.Run())
}
