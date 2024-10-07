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
	"strings"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"v2ray.com/core/common/uuid"
)

var RDB *redis.Client
var ctx = context.Background()
var bt tg.Bot
var cc *grpc.ClientConn

type Userinfo struct {
	Usedbwpretty string
	Usedbw       int
	UserId       int
}

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

func procIncome(update tg.Update, cc *grpc.ClientConn) {

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
				bt.SendMessage(fmt.Sprintf("@naharlo \n- /start\n- /revoke \n\nhttps://choskosh.cfd/stat.html?uuid=%s", new_uuid.String()), update.Message.From.Id)

			} else {
				if userUUid == "BlOCKED" {
					bt.SendMessage("you are blocked", update.Message.From.Id)
				} else {
					used := v2rpc.GetUserStat(strconv.Itoa(update.Message.From.Id), cc)
					bt.SendMessage(fmt.Sprintf("uuid: %s \n\ntransfered: %s", userUUid, ByteCountSI(int64(used))), update.Message.From.Id)

				}

			}

		}
	case "/revoke":
		{
			userUUid, _ := RDB.Get(ctx, strconv.Itoa(update.Message.From.Id)).Result()
			if userUUid == "BLOCKED" {
				bt.SendMessage("you are blocked", update.Message.From.Id)

			} else {
				v2rpc.RemoveUser(strconv.Itoa(update.Message.From.Id), cc)
				new_uuid := uuid.New()
				_, err := v2rpc.Adduser(new_uuid.String(), strconv.Itoa(update.Message.From.Id), cc)
				if err != nil {
					bt.SendMessage("failed", update.Message.From.Id)
					log.Print("err: ", err)
					return
				}
				RDB.Set(ctx, strconv.Itoa(update.Message.From.Id), new_uuid.String(), 0)
				bt.SendMessage(fmt.Sprintf("new uuid generated \n\nhttps://choskosh.cfd/stat.html?uuid=%s", new_uuid.String()), update.Message.From.Id)
			}
		}

	}
}
func startup(cc *grpc.ClientConn) {
	iter := RDB.Scan(ctx, 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		userUUid, err := RDB.Get(ctx, iter.Val()).Result()
		if err != nil {
			log.Print("err startup iter: ", err)
		}
		if userUUid != "BLOCKED" {
			v2rpc.Adduser(userUUid, iter.Val(), cc)
		}
	}

}

func main() {
	tgToken := flag.String("tg", "bot123", "telegram token")
	redisaddr := flag.String("rdis", "redis:6379", "redis addr")
	v2raygrpc := flag.String("v2", "v2ray:8080", "v2ray grpc endpoint address")
	flag.Parse()

	bt = tg.Bot{
		Token: *tgToken,
	}

	RDB = redis.NewClient(&redis.Options{
		Addr:     *redisaddr,
		Password: "",
		DB:       0,
	})

	cc = v2rpc.GetGrpcConn(*v2raygrpc)

	startup(cc)

	http.HandleFunc(fmt.Sprintf("/v2api/%s/", *tgToken), func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			{
				var up tg.Update
				err := json.NewDecoder(r.Body).Decode(&up)
				if err != nil {
					fmt.Printf("err: %v\n", err)
				}
				w.WriteHeader(http.StatusOK)
				go procIncome(up, cc)
			}
		default:
			{
				urlPath := r.URL.Path

				parts := strings.Split(urlPath, "/")
				if len(parts) != 5 {
					var usrlis []Userinfo
					iter := RDB.Scan(ctx, 0, "*", 0).Iterator()
					for iter.Next(ctx) {
						used := v2rpc.GetUserStat(iter.Val(), cc)
						if used != 0 {
							userid, _ := strconv.Atoi(iter.Val())
							uo := Userinfo{Usedbwpretty: ByteCountSI(int64(used)), Usedbw: int(used), UserId: userid}
							usrlis = append(usrlis, uo)
						}

					}
					j, _ := json.Marshal(usrlis)
					w.Write(j)
					return
				}
				cmd := parts[3]
				uid := parts[4]
				switch cmd {
				case "block":
					{
						_, err := RDB.Get(ctx, uid).Result()
						if err != nil {
							w.Write([]byte(err.Error()))
							return
						}
						_, err = v2rpc.RemoveUser(uid, cc)
						RDB.Set(ctx, uid, "BLOCKED", 0)
						w.Write([]byte(err.Error()))
					}
				case "unblock":
					{
						uidint, err := strconv.Atoi(uid)
						if err != nil {
							w.Write([]byte(err.Error()))
							return
						}
						new_uuid := uuid.New()
						_, err = v2rpc.Adduser(new_uuid.String(), uid, cc)
						if err != nil {
							bt.SendMessage("failed to unblock", uidint)
							w.Write([]byte(err.Error()))
							return
						}
						RDB.Set(ctx, uid, new_uuid.String(), 0)
						bt.SendMessage(fmt.Sprintf("new uuid generated \n\nhttps://choskosh.cfd/stat.html?uuid=%s", new_uuid.String()), uidint)
						w.Write([]byte("unblocked"))
					}
				}

			}
		}

	})

	http.ListenAndServe("0.0.0.0:2020", nil)
}
