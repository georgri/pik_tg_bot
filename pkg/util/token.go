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
var initBotToken sync.Once

func GetBotToken() string {
	initBotToken.Do(func() {
		envType := GetEnvType()
		if envType == EnvTypeDev || envType == EnvTypeTesting {
			botToken = GetTestBotToken()
			return
		}

		// read the TokenFile file. If fail => return init as test bot token
		content, err := os.ReadFile(TokenFile)
		if err != nil || len(content) == 0 {
			botToken = GetTestBotToken()
			return
		}

		botToken = strings.TrimSpace(string(content))
	})

	return botToken
}
