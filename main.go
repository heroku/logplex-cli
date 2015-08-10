package main

import (
	"fmt"
	"log"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/franela/goreq"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Endpoint    string `envconfig:"LOGPLEX_ENDPOINT"`
	AuthKey     string `envconfig:"LOGPLEX_AUTH_KEY"`
	HerokuCloud string `envconfig:"HEROKU_CLOUD"`
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
			config.Endpoint = "https://ops.dev.logplex.io"
		case "production":
			config.Endpoint = "https://logs-api.heroku.com"
		default:
			log.Fatalf("Probably a devcloud, TODO")
		}
	}

	if config.AuthKey == "" {
		log.Fatalf("$LOGPLEX_AUTH_KEY not set; you can find it in hgetall cred:api of logplex_config_redis")
	}
}

func main() {
	usage := `Logplex CLI.

	Usage:
		logplex-cli channel create <name> <token>...
		logplex-cli channel destroy <name>
	`

	arguments, err := docopt.Parse(usage, nil, true, "Logplex CLI", false)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	log.Printf("Arguments => %+v\n", arguments)

	readConfig()

	log.Printf("Config => %+v\n", config)

	if _, ok := arguments["channel"]; ok {
		if _, ok := arguments["create"]; ok {
			// create
			str := arguments["<name>"].(string)
			tokens := arguments["<token>"].([]string)

			err := createChannel(&Channel{Name: str, Tokens: tokens})
			if err != nil {
				log.Fatalf("ERR: %v\n", err)
			}

		} else if _, ok := arguments["destroy"]; ok {
			// destroy
			log.Printf("channel destroy %v\n", arguments["<token>"])
		}
	}

}

func createChannel(payload *Channel) error {
	// TODO: possibly ignore request certificates
	// https://github.com/heroku/heroku-cli/commit/75403de1a0d581e1eb9acfffe9ab0443e3f36a38
	req := goreq.Request{
		Method:      "POST",
		Uri:         fmt.Sprintf("%s/channels", config.Endpoint),
		Body:        payload,
		ContentType: "application/json",
	}.WithHeader("Authorization", fmt.Sprintf("Basic %s", config.AuthKey))

	response, err := req.Do()
	if err == nil {
		text, err := response.Body.ToString()
		if err != nil {
			return err
		}
		fmt.Printf("Response (%v): %v\n", response.Status, text)
		defer response.Body.Close()
	}

	return err
}

type Channel struct {
	Name   string   `json:"name"`
	Tokens []string `json:"tokens"`
}
