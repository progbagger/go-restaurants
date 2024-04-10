package db

import (
	"common"
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

func CreateClient(CACert []byte) (*elasticsearch.Client, error) {
	credentials := getCredentials()
	return elasticsearch.NewClient(elasticsearch.Config{
		Password:  credentials[elasticPassword],
		Username:  credentials[elasticUser],
		Addresses: []string{credentials[elasticUrl]},
		CACert:    CACert,
	})
}
