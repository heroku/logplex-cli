package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/franela/goreq"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Endpoint    string `envconfig:"LOGPLEX_ENDPOINT"`
	AuthKey     string `envconfig:"LOGPLEX_AUTH_KEY" required:"true"`
	HerokuCloud string `envconfig:"HEROKU_CLOUD" required:"true"`
}

var config Config

func init() {
	if err := envconfig.Process("logplex", &config); err != nil {
		log.Fatalf("ERR: %v", err)
	}

	if config.Endpoint == "" {
		switch config.HerokuCloud {
		case "ops":
			config.Endpoint = "https://logs-api.herokai.com"
		case "production":
			config.Endpoint = "https://logs-api.heroku.com"
		default:
			log.Fatalf("Probably a devcloud, TODO")
		}
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

	log.Printf("%+v\n", arguments)

	if _, ok := arguments["channel"]; ok {
		log.Println("channel")

		if _, ok := arguments["create"]; ok {
			// create
			str := arguments["<name>"].(string)
			tokens := arguments["<token>"].([]string)

			err := createChannel(&Channel{Name: str, Tokens: tokens})
			log.Printf("ERR: %v", err)

		} else if _, ok := arguments["destroy"]; ok {
			// destroy
			log.Printf("channel destroy %v", arguments["<token>"])
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

	response, error := req.Do()
	if error == nil {
		io.Copy(os.Stdout, response.Body)
		defer response.Body.Close()
	}

	return error
}

type Channel struct {
	Name   string   `json:"name"`
	Tokens []string `json:"tokens"`
}