package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"mmd/v2mngo/tg"
	"mmd/v2mngo/v2rpc"
	"net/http"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	"v2ray.com/core/common/uuid"
)

type User struct {
	Uuid   string
	Active bool
}

func procIncome(update tg.Update, tk string, rdcli *redis.Client) {
	bt := tg.Bot{
		Token: tk,
	}

	switch update.Message.Text {
	case "/start":
		{
			_, err := rdcli.HGetAll(context.Background(),
				strconv.Itoa(update.Message.From.Id),
			).Result()

			fmt.Printf("err: %v\n", err)
			if err != nil {
				fmt.Printf("val is empty \n")
				uid := uuid.New()
				rdcli.HSet(context.Background(), strconv.Itoa(update.Message.From.Id), User{
					Uuid:   uid.String(),
					Active: false,
				},
				)
			}

			bt.SendMessage("@naharlo \n- /stat", update.Message.From.Id)
		}
	case "/stat":
		{
			var user User
			err := rdcli.HGetAll(context.Background(),
				strconv.Itoa(update.Message.From.Id),
			).Scan(&user)
			if err != nil {
				fmt.Printf("err: %v\n", err)
			}

			bt.SendMessage(fmt.Sprintf("you - uuid: %s \n Active ", user.Uuid), update.Message.From.Id)
		}

	}
}

func main() {
	hookp := flag.String("tg", "hook", "telegram hook endpoint")
	admin_endpoint := flag.String("k", "kkdkkd", "admin endpoint key endpoint")

	flag.Parse()

	bt := tg.Bot{
		Token: *hookp,
	}
	cc := v2rpc.GetGrpcConn("127.0.0.1:8081")
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

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
				go procIncome(up, *hookp, redisClient)
			}
		default:
			{
				w.Write([]byte("this is good "))
			}
		}

	})

	http.HandleFunc(fmt.Sprintf("/v2api/%s/", *admin_endpoint), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		p := strings.Split(r.URL.Path, "/")

		var mail string
		if len(p) >= 3 {
			mail = p[3]
		} else {
			mail = ""
		}

		if len(p) <= 2 {
			w.Write([]byte(`{"endpointOk": true}`))
			return
		}

		switch p[2] {
		case "a":
			{
				usrUuid, err := redisClient.HGet(context.Background(), mail, "uuid").Result()
				if err != nil {
					w.Write([]byte(`{"ok": false}`))
					return
				}
				_, err = v2rpc.Adduser(usrUuid, mail, cc)
				if err != nil {
					w.Write([]byte(`{"ok": false}`))
					return
				}
				redisClient.HSet(context.Background(), mail, User{Uuid: usrUuid, Active: true})
				id, _ := strconv.Atoi(mail)
				bt.SendMessage(fmt.Sprintf("you are activated, configs are valid at http://ar3642.top/stat.html?uuid=%s", usrUuid), id)

			}
		case "da":
			{
				_, err := v2rpc.RemoveUser(mail, cc)
				if err != nil {
					w.Write([]byte(`{"ok": false}`))
					return
				}
				id, _ := strconv.Atoi(mail)
				bt.SendMessage("you are not active any more", id)

			}
		case "ls":
			{
				keys, err := redisClient.Keys(context.Background(), "*").Result()
				if err != nil {
					w.Write([]byte(`{"ok": false}`))
					return
				}
				bj, _ := json.Marshal(map[string][]string{"keys": keys})
				w.Write(bj)
				return
			}

		}

		w.Write([]byte(`{"ok":true}`))

	})

	http.HandleFunc("/v2api/user/", func(w http.ResponseWriter, r *http.Request) {

		// /v2api/user/<mail>/<uuid>

		p := strings.Split(r.URL.Path, "/")
		userUuid := p[len(p)-1]
		mail := p[len(p)-2]
		usage := v2rpc.GetUserStat(mail, cc)

		var usr User
		err := redisClient.HGet(context.Background(), mail, "active").Scan(&usr)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if usr.Uuid != userUuid {
			w.WriteHeader(405)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{
			"uuid":%s, 
			"active":%v, 
			"stat":%d
		}`, usr.Uuid, usr.Active, usage)))

	})

	http.ListenAndServe("0.0.0.0:4040", nil)
}
