package main

import (
	"os"
	"regexp"

	"github.com/joho/godotenv"
	"github.com/modern-pet/flarebot/googledocs"
	"github.com/modern-pet/flarebot/slack"
)

// #flare-179-foo-bar --> #flare-179
var channelNameRegexp = regexp.MustCompile("^([^-]+-[^-]+)(?:-.+)")
var flareChannelNamePrefix *regexp.Regexp

// Save slack history doc IDs in a cache for more efficient lookups
var slackHistoryDocCache = map[string]string{}

func main() {
	godotenv.Load()

	flareChannelNamePrefix = regexp.MustCompile("flare-")

	// Google Docs service
	googleDocsServer, err := googledocs.NewGoogleDocsServerWithServiceAccount(os.Getenv("GOOGLE_FLAREBOT_SERVICE_ACCOUNT_CONF"))
	if err != nil {
		panic(err)
	}
	googleDomain := os.Getenv("GOOGLE_DOMAIN")

	googleFlareDocID := os.Getenv("GOOGLE_TEMPLATE_DOC_ID")
	googleSlackHistoryDocID := os.Getenv("GOOGLE_TEMPLATE_SLACK_HISTORY_DOC_ID")

	// domain := os.Getenv("SLACK_DOMAIN")
	username := os.Getenv("SLACK_USERNAME")
	expectedChannel := os.Getenv("SLACK_CHANNEL")

	// Instantiate slack socket mode client
	slackClient, err := slack.NewSlackClient(username, expectedChannel, googleDocsServer, googleDomain, googleFlareDocID, googleSlackHistoryDocID)
	if err != nil {
		panic(err)
	}

	panic(slackClient.Client.Run())

	// client.Respond(takingLeadCommand.regexp, func(msg *slack.Message, params [][]string) {
	// 	author, _ := msg.AuthorUser()

	// 	client.Send("working on assigning incident lead....", msg.Channel)

	// 	client.Send(fmt.Sprintf("Oh Captain My Captain! @%s is now incident lead. Please confirm all actions with them.", author.Name), msg.Channel)
	// })

	// client.Respond(flareMitigatedCommand.regexp, func(msg *slack.Message, params [][]string) {
	// 	// notify the main flares channel
	// 	client.Send("Flare has been mitigated", expectedChannel)
	// })

	// client.Respond(notAFlareCommand.regexp, func(msg *slack.Message, params [][]string) {
	// 	// notify the main flares channel
	// 	client.Send("turns out this is not a Flare", expectedChannel)
	// })

	// client.Respond(helpCommand.regexp, func(msg *slack.Message, params [][]string) {
	// 	sendHelpMessage(client, msg.Channel, (msg.Channel == expectedChannel))
	// })

	// client.Respond(helpAllCommand.regexp, func(msg *slack.Message, param [][]string) {
	// 	client.Send("Commands Available in the #flares channel:", msg.Channel)
	// 	sendCommandsHelpMessage(client, msg.Channel, mainChannelCommands)
	// 	client.Send("Commands Available in a single Flare channel:", msg.Channel)
	// 	sendCommandsHelpMessage(client, msg.Channel, flareChannelCommands)
	// 	client.Send("Commands Available in other channels:", msg.Channel)
	// 	sendCommandsHelpMessage(client, msg.Channel, otherChannelCommands)
	// })

	// // fallback response saying "I don't understand"
	// client.Respond(".*", func(msg *slack.Message, params [][]string) {
	// 	// should be taking commands here, and didn't understand
	// 	client.Send(`I'm sorry, I didn't understand that command.
	// 		To fire a flare: @flarebot fire a flare <p0|p1|p2> [pre-emptive|retroactive] <problem>
	// 		For other commands: @flarebot help [all]
	// 	`, msg.Channel)
	// })

	// panic(client.Run())
}
