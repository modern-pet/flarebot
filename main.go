package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/joho/godotenv"
	"github.com/modern-pet/flarebot/aws"
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

	googleDocsServerConfig := os.Getenv("GOOGLE_FLAREBOT_SERVICE_ACCOUNT_CONF")
	googleDomain := os.Getenv("GOOGLE_DOMAIN")
	googleFlareDocID := os.Getenv("GOOGLE_TEMPLATE_DOC_ID")
	googleSlackHistoryDocID := os.Getenv("GOOGLE_TEMPLATE_SLACK_HISTORY_DOC_ID")
	username := os.Getenv("SLACK_USERNAME")
	expectedChannel := os.Getenv("SLACK_CHANNEL")

	// Google Docs service
	googleDocsServer, err := googledocs.NewGoogleDocsServerWithServiceAccount(googleDocsServerConfig)
	if err != nil {
		panic(fmt.Errorf("Failed to initialize google docs server with error: %s", err))
	}
	// AWS Client to keep track of flare channel IDs
	if err = aws.InitializeAWSClient(); err != nil {
		panic(fmt.Errorf("Failed to initialize aws client with error: %s", err))
	}

	// Instantiate slack socket mode client
	slackClient, err := slack.NewSlackClient(username, expectedChannel, googleDocsServer, googleDomain, googleFlareDocID, googleSlackHistoryDocID)
	if err != nil {
		panic(err)
	}

	panic(slackClient.Client.Run())
}
