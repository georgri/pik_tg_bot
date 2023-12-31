package telegrambot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	TestChatID = -1002057808675

	messageCharLimit = 4000
)

// An example of how to send message with test bot:
// https://api.telegram.org/bot6819149165:AAEQWnUotV_YsGS7EPaNbUKZpcvKhsmOgNg/sendMessage?chat_id=-1002057808675&text=hello_friend
// i.e. https://api.telegram.org/bot{token}/sendMessage?chat_id={chat_id}&text={text}

func SendTestMessage(text string) error {
	return SendMessage(TestChatID, text)
}

func SendMessage(chatID int64, text string) error {
	token := GetBotToken()

	chunks := SplitTextIntoSendableChunks(text)

	for _, msg := range chunks {
		err := SendMessageWithToken(token, chatID, msg)
		if err != nil {
			return err
		}
	}

	return nil
}

type SendResponse struct {
	OK bool `json:"ok"`
}

func SendMessageWithToken(token string, chatID int64, text string) error {

	sendMessageUrl := fmt.Sprintf("https://api.telegram.org/bot%v/sendMessage", token)

	values := url.Values{
		"chat_id":                  []string{fmt.Sprintf("%v", chatID)},
		"text":                     []string{text},
		"parse_mode":               []string{"HTML"},
		"disable_web_page_preview": []string{"True"},
	}
	// post http request
	resp, err := http.PostForm(sendMessageUrl, values)
	if err != nil {
		return err
	}

	if resp.ContentLength < 0 {
		return fmt.Errorf("can't read body because content len < 0: %v", resp.Request.URL)
	}

	// example of response:
	// {"ok":true,"result":{"message_id":5,"sender_chat":{"id":-1002057808675,"title":"Pik checker bot tester","username":"pik_checker_bot_tester","type":"channel"},"chat":{"id":-1002057808675,"title":"Pik checker bot tester","username":"pik_checker_bot_tester","type":"channel"},"date":1701824824,"text":"hello_friend"}}
	body := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(body)
	if err != nil {
		return fmt.Errorf("error while reading Body: %v", resp.Request.URL)
	}

	sendResponse := &SendResponse{}
	err = json.Unmarshal(body, sendResponse)
	if err != nil {
		return fmt.Errorf("error while unmarshalling Body: %v", string(body))
	}

	if !sendResponse.OK {
		return fmt.Errorf("send response is not OK: %v", string(body))
	}

	return nil
}

func SplitTextIntoSendableChunks(text string) []string {
	lines := strings.Split(text, "\n")

	res := make([]string, 0, 5)
	var bufFrom, bufSize int
	for i, line := range lines {
		if bufSize+len(line) > messageCharLimit {
			res = append(res, strings.Join(lines[bufFrom:i], "\n"))
			bufFrom = i
			bufSize = 0
		}
		bufSize += len(line) + 1
	}

	res = append(res, strings.Join(lines[bufFrom:], "\n"))

	return res
}
