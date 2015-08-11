package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"encoding/json"
	"github.com/docopt/docopt-go"
	"github.com/franela/goreq"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Debug       bool   `envconfig:"DEBUG"`
	Endpoint    string `envconfig:"LOGPLEX_ENDPOINT"`
	AuthKey     string `envconfig:"LOGPLEX_AUTH_KEY"`
	HerokuCloud string `envconfig:"HEROKU_CLOUD"`
	SslInsecure bool   `envconfig:"SSL_INSECURE"`
}

var config Config

func readConfig() {
	if err := envconfig.Process("logplex", &config); err != nil {
		log.Fatalf("ERR: not all environment vars are set (%v)", err)
	}

	if config.Endpoint == "" {
		if config.HerokuCloud == "" {
			log.Fatalf("Either $HEROKU_CLOUD or $LOGPLEX_ENDPOINT must be set")
		}
		switch config.HerokuCloud {
		case "ops":
			config.Endpoint = "https://logs-api.herokai.com"
		case "production":
			config.Endpoint = "https://logs-api.heroku.com"
		default:
			config.Endpoint = fmt.Sprintf("https://logplex-api-ssl.ssl.%s.herokudev.com", config.HerokuCloud)
			config.SslInsecure = true
		}
	}

	if config.AuthKey == "" {
		log.Fatalf("$LOGPLEX_AUTH_KEY is not set; retrieve it using `ion-client config:get -a logplex LOGPLEX_AUTH_KEY`")
	}

	if config.SslInsecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		goreq.DefaultClient = &http.Client{Transport: tr}
	}
}

func IsSubcommand(arguments map[string]interface{}, group, command string) bool {
	invoked, ok := arguments[group].(bool)
	if ok && invoked {
		invoked, ok = arguments[command].(bool)
		return ok && invoked
	}
	return false
}

func main() {
	usage := `Logplex CLI.

Usage:
	logplex-cli channel create <name> <token>...
	logplex-cli channel destroy <channelId>
  logplex-cli drain add <channelId> <drainUrl>
  logplex-cli drain remove <channelId> <drainId>
	`

	arguments, err := docopt.Parse(usage, nil, true, "Logplex CLI", false)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	readConfig()

	if config.Debug {
		log.Printf("Config => %+v\n", config)
		log.Printf("Arguments => %+v\n", arguments)
	}

	value, err := runCommand(arguments)
	if err != nil {
		log.Fatal(err.Error())
	} else {
		// Dumping JSON for scriptability. Ideally should have --json argument.
		bytes, err := json.Marshal(value)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", string(bytes))
	}
}

func runCommand(arguments map[string]interface{}) (interface{}, error) {
	dummyValue := map[string]string{}
	switch {
	case IsSubcommand(arguments, "channel", "create"):
		str := arguments["<name>"].(string)
		tokens := arguments["<token>"].([]string)
		return createChannel(&ChannelRequest{Name: str, Tokens: tokens})
	case IsSubcommand(arguments, "channel", "destroy"):
		channelId := arguments["<channelId>"].(string)
		return dummyValue, destroyChannel(channelId)
	case IsSubcommand(arguments, "drain", "add"):
		channelId := arguments["<channelId>"].(string)
		drainUrl := arguments["<drainUrl>"].(string)
		return addDrain(channelId, drainUrl)
	case IsSubcommand(arguments, "drain", "remove"):
		channelId := arguments["<channelId>"].(string)
		drainId := arguments["<drainId>"].(string)
		return dummyValue, removeDrain(channelId, drainId)
	}
	log.Fatalf("unreachable")
	return dummyValue, nil
}

//
// channel:create
//

func createChannel(payload *ChannelRequest) (*ChannelResponse, error) {
	// TODO: possibly ignore request certificates
	// https://github.com/heroku/heroku-cli/commit/75403de1a0d581e1eb9acfffe9ab0443e3f36a38
	req := goreq.Request{
		Method:      "POST",
		Uri:         fmt.Sprintf("%s/channels", config.Endpoint),
		Body:        payload,
		ContentType: "application/json",
	}.WithHeader("Authorization", fmt.Sprintf("Basic %s", config.AuthKey))

	response, err := req.Do()
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 201 {
		return nil, fmt.Errorf("Unsuccessful response (%v) from logplex", response.Status)
	}

	var channelResponse ChannelResponse
	err = response.Body.FromJsonTo(&channelResponse)
	if err != nil {
		return nil, err
	}

	return &channelResponse, err
}

type ChannelRequest struct {
	Name   string   `json:"name"`
	Tokens []string `json:"tokens"`
}

type ChannelResponse struct {
	ChannelId int               `json:"channel_id"`
	Tokens    map[string]string `json:"tokens"`
}

//
// channel:destroy
//

func destroyChannel(channelId string) error {
	req := goreq.Request{
		Method: "DELETE",
		Uri:    fmt.Sprintf("%s/v2/channels/%s", config.Endpoint, channelId),
	}.WithHeader("Authorization", fmt.Sprintf("Basic %s", config.AuthKey))

	response, err := req.Do()
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("Unsuccessful response (%v) from logplex", response.Status)
	}
	return nil
}

// drain:add

func addDrain(channelId, drainUrl string) (*DrainResponse, error) {
	var payload struct {
		Url string `json:"url"`
	}
	payload.Url = drainUrl

	req := goreq.Request{
		Method:      "POST",
		Uri:         fmt.Sprintf("%s/v2/channels/%s/drains", config.Endpoint, channelId),
		Body:        payload,
		ContentType: "application/json",
	}.WithHeader("Authorization", fmt.Sprintf("Basic %s", config.AuthKey))

	response, err := req.Do()
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 201 {
		return nil, fmt.Errorf("Unsuccessful response (%v) from logplex", response.Status)
	}

	var drainResponse DrainResponse
	err = response.Body.FromJsonTo(&drainResponse)
	if err != nil {
		return nil, err
	}

	return &drainResponse, err
}

type DrainResponse struct {
	Id    int    `json:"id"`
	Token string `json:"token"`
	Url   string `json:"url"`
}

// drain:remove

func removeDrain(channelId, drainId string) error {
	req := goreq.Request{
		Method: "DELETE",
		Uri:    fmt.Sprintf("%s/v2/channels/%s/drains/%s", config.Endpoint, channelId, drainId),
	}.WithHeader("Authorization", fmt.Sprintf("Basic %s", config.AuthKey))

	response, err := req.Do()
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("Unsuccessful response (%v) from logplex", response.Status)
	}
	return nil
}
