package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mmd/v2mngo/tg"
	"mmd/v2mngo/v2rpc"
	"net/http"
	"strconv"

	"google.golang.org/grpc"
	"v2ray.com/core/common/uuid"
)

var Users = make(map[int]string)

func procIncome(update tg.Update, tk string, cc *grpc.ClientConn) {
	bt := tg.Bot{
		Token: tk,
	}

	switch update.Message.Text {
	case "/start":
		{

			user_uuid, ok := Users[update.Message.From.Id]

			if ok {
				used := v2rpc.GetUserStat(strconv.Itoa(update.Message.From.Id), cc)
				bt.SendMessage(fmt.Sprintf("uuid: %s \n\n transfered: %d", user_uuid, used), update.Message.From.Id)

			} else {
				new_uuid := uuid.New()
				Users[update.Message.From.Id] = new_uuid.String()
				_, err := v2rpc.Adduser(new_uuid.String(), strconv.Itoa(update.Message.From.Id), cc)
				if err != nil {
					bt.SendMessage("failed", update.Message.From.Id)
					log.Print("err: ", err)
					return
				}
				bt.SendMessage(fmt.Sprintf("@naharlo \n\nuuid: %s", new_uuid.String()), update.Message.From.Id)

			}

		}

	}
}

func main() {
	hookp := flag.String("tg", "hook", "telegram hook endpoint")

	flag.Parse()

	cc := v2rpc.GetGrpcConn("127.0.0.1:8080")

	http.HandleFunc(fmt.Sprintf("/v2api/%s", *hookp), func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			{
				var up tg.Update
				err := json.NewDecoder(r.Body).Decode(&up)
				if err != nil {
					fmt.Printf("err: %v\n", err)
				}
				w.WriteHeader(http.StatusOK)
				go procIncome(up, *hookp, cc)
			}
		default:
			{
				w.Write([]byte("200 Ok!"))
			}
		}

	})

	http.ListenAndServe("127.0.0.1:2020", nil)
}
