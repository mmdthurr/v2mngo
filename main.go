package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mmd/v2mngo/tg"
	"mmd/v2mngo/v2rpc"
	"net/http"
	"strconv"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"v2ray.com/core/common/uuid"
)

var RDB *redis.Client
var ctx = context.Background()

func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

func procIncome(update tg.Update, tk string, cc *grpc.ClientConn) {
	bt := tg.Bot{
		Token: tk,
	}

	switch update.Message.Text {
	case "/start":
		{
			userUUid, err := RDB.Get(ctx, strconv.Itoa(update.Message.From.Id)).Result()
			if err != nil {
				new_uuid := uuid.New()
				_, err := v2rpc.Adduser(new_uuid.String(), strconv.Itoa(update.Message.From.Id), cc)
				if err != nil {
					bt.SendMessage("failed", update.Message.From.Id)
					log.Print("err: ", err)
					return
				}
				RDB.Set(ctx, strconv.Itoa(update.Message.From.Id), new_uuid.String(), 0)
				bt.SendMessage(fmt.Sprintf("@naharlo - /start \n\nhttps://ar3642.top/stat.html?uuid=%s", new_uuid.String()), update.Message.From.Id)

			} else {
				used := v2rpc.GetUserStat(strconv.Itoa(update.Message.From.Id), cc)
				bt.SendMessage(fmt.Sprintf("uuid: %s \n\ntransfered: %s", userUUid, ByteCountSI(int64(used))), update.Message.From.Id)

			}

		}

	}
}

func main() {
	tgToken := flag.String("tg", "bot123", "telegram token")
	flag.Parse()

	RDB = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	cc := v2rpc.GetGrpcConn("127.0.0.1:8080")

	http.HandleFunc(fmt.Sprintf("/v2api/%s", *tgToken), func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			{
				var up tg.Update
				err := json.NewDecoder(r.Body).Decode(&up)
				if err != nil {
					fmt.Printf("err: %v\n", err)
				}
				w.WriteHeader(http.StatusOK)
				go procIncome(up, *tgToken, cc)
			}
		default:
			{
				w.Write([]byte("okeymokey"))
			}
		}

	})

	http.ListenAndServe("127.0.0.1:2020", nil)
}
