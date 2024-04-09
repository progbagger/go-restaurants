package client

import (
	"args"
	"common"
	"log"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
)

func parseEnv(keys ...string) map[string]common.Pair[string, bool] {
	result := make(map[string]common.Pair[string, bool], len(keys))

	for _, key := range keys {
		value, isPresent := os.LookupEnv(key)
		result[key] = common.Pair[string, bool]{First: value, Second: isPresent}
	}

	return result
}

const (
	elasticUser     string = "ELASTIC_USER"
	elasticPassword string = "ELASTIC_PASSWORD"
	elasticUrl      string = "ELASTIC_URL"

	defaultElasticUser     string = "elastic"
	defaultElasticPassword string = "elastic"
	defaultElasticUrl      string = "https://localhost:9200"
)

func getDefaultArgs() []args.Arg {
	return []args.Arg{
		{
			Name:         "cacert",
			Description:  "path to the HTTPS certificate",
			DefaultValue: "",
			Required:     true,
		},
	}
}

func getCredentials() map[string]string {
	env := parseEnv(elasticPassword, elasticUrl, elasticUser)
	passwordEntry, urlEntry, userEntry := env[elasticPassword], env[elasticUrl], env[elasticUser]

	result := make(map[string]string, len(env))

	if !passwordEntry.Second {
		result[elasticPassword] = defaultElasticPassword
	} else {
		result[elasticPassword] = passwordEntry.First
	}

	if !urlEntry.Second {
		result[elasticUrl] = defaultElasticUrl
	} else {
		result[elasticUrl] = urlEntry.First
	}

	if !userEntry.Second {
		result[elasticUser] = defaultElasticUser
	} else {
		result[elasticUser] = userEntry.First
	}

	return result
}

func CreateClient() (*elasticsearch.Client, error) {
	log.SetFlags(log.Lshortfile)

	// acquiring path to the cacert
	parsedArgs, _, err := args.ParseArgs(getDefaultArgs()...)
	if err != nil {
		return nil, err
	}
	caCertFile := parsedArgs["cacert"].(string)

	credentials := getCredentials()
	caCert, err := os.ReadFile(caCertFile)
	if err != nil {
		return nil, err
	}

	return elasticsearch.NewClient(elasticsearch.Config{
		Password:  credentials[elasticPassword],
		Username:  credentials[elasticUser],
		Addresses: []string{credentials[elasticUrl]},
		CACert:    caCert,
	})
}
