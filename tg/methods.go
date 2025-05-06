package tg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type Bot struct {
	Token string
}

func (bt Bot) SendMessage(msg string, chat_id int) {

	tg_json, _ := json.Marshal(map[string]any{
		"chat_id":    chat_id,
		"parse_mode": "HTML",
		"text":       msg,
	})

	os.Setenv("HTTP_PROXY", "http://127.0.0.1:1060")
	_, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", bt.Token), "application/json", bytes.NewBuffer(tg_json))
	if err != nil {
		log.Printf("tg: sendmsg: err: %s", err)

	}

}
