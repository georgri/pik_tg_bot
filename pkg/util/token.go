package util

import (
	"os"
	"strings"
	"sync"
)

const (
	TestBotToken = "6819149165:AAEQWnUotV_YsGS7EPaNbUKZpcvKhsmOgNg"

	TokenFile = "bot.token"
)

func GetTestBotToken() string {
	return TestBotToken
}

var botToken string
var botTokenEnvType EnvType
var botTokenInit bool
var botTokenMu sync.Mutex

func GetBotToken() string {
	envType := GetEnvType()

	// NOTE: envtype comes from a flag defaulting to "dev".
	// Some packages used to call GetBotToken() at init-time (before flag.Parse()),
	// which would permanently cache the test token even when envtype later becomes "prod".
	// To make this robust, we recompute if envtype has changed since last load.
	botTokenMu.Lock()
	defer botTokenMu.Unlock()

	if botTokenInit && botTokenEnvType == envType {
		return botToken
	}

	botTokenInit = true
	botTokenEnvType = envType

	if envType == EnvTypeDev || envType == EnvTypeTesting {
		botToken = GetTestBotToken()
		return botToken
	}

	// read the TokenFile file. If fail => return init as test bot token
	content, err := os.ReadFile(TokenFile)
	if err != nil || len(content) == 0 {
		botToken = GetTestBotToken()
		return botToken
	}

	botToken = strings.TrimSpace(string(content))
	return botToken
}

func GetBotUsername() string {
	envType := GetEnvType()
	if envType == EnvTypeDev || envType == EnvTypeTesting {
		return "PikCheckerTestBot"
	}
	return "pik_checker_bot"
}
