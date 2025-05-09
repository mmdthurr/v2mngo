package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mmd/v2mngo/db"
	"mmd/v2mngo/tg"
	"mmd/v2mngo/v2rpc"
	"net/http"
	"net/url"
	"strconv"

	"google.golang.org/grpc"
	"gorm.io/gorm"
	"v2ray.com/core/common/uuid"
)

var bt tg.Bot
var cc *grpc.ClientConn
var datab *gorm.DB

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

func HandleUpdate(update tg.Update, cc *grpc.ClientConn, domain string, name string) {

	switch update.Message.Text {
	case "/start":
		{
			user := db.User{}
			result := datab.FirstOrCreate(&user, db.User{TgId: update.Message.From.Id})

			if result.RowsAffected == 1 {
				log.Println("new user created")
				new_uuid := uuid.New()
				_, err := v2rpc.Adduser(new_uuid.String(), strconv.Itoa(update.Message.From.Id), cc)
				if err != nil {
					bt.SendMessage("502 failed", update.Message.From.Id)
					log.Printf("main: HandleUpdated: v2rpcadduser: err: %s\n", err)
					return
				}
				user.UUID = new_uuid.String()
				datab.Save(&user)

				//wellcome message
				bt.SendMessage(fmt.Sprintf("mmdta.ir \n@naharlo \n- /start\n- /revoke\n\nhttps://%s/stat.html?uuid=%s&srv=%s", domain, new_uuid.String(), name), update.Message.From.Id)

			} else {
				if user.Blocked == true {
					bt.SendMessage(fmt.Sprintf("you are blocked due to %s \n", user.LastBlockedReason), update.Message.From.Id)
				} else {
					used := v2rpc.GetUserStat(strconv.Itoa(update.Message.From.Id), cc)
					bt.SendMessage(fmt.Sprintf("uuid: %s \n\ntransfered: %s", user.UUID, ByteCountSI(int64(used))), update.Message.From.Id)
				}
			}

		}
	case "/revoke":
		{
			user := db.User{}
			result := datab.FirstOrCreate(&user, db.User{TgId: update.Message.From.Id})

			if result.RowsAffected == 1 {
				bt.SendMessage("mmdta.ir \n@naharlo \n- /start\n- /revoke\n\n", update.Message.From.Id)
			}
			if user.Blocked {
				return
			} else {
				v2rpc.RemoveUser(strconv.Itoa(update.Message.From.Id), cc)
				new_uuid := uuid.New()
				_, err := v2rpc.Adduser(new_uuid.String(), strconv.Itoa(update.Message.From.Id), cc)
				if err != nil {
					bt.SendMessage("502 failed", update.Message.From.Id)
					log.Print("HandleUpdate: revoke: err: ", err)
					return
				}
				user.UUID = new_uuid.String()
				// update
				datab.Save(&user)
				bt.SendMessage(fmt.Sprintf("new uuid generated \n\nhttps://%s/stat.html?uuid=%s&srv=%s", domain, new_uuid.String(), name), update.Message.From.Id)
			}
		}

	}
}

func startup(cc *grpc.ClientConn) {
	var users []db.User
	datab.Where("blocked = ?", false).Find(&users)

	for _, user := range users {

		_, err := v2rpc.Adduser(user.UUID, strconv.Itoa(int(user.TgId)), cc)
		if err != nil {
			log.Printf("this is user: %s", err)
		}
	}

}

func main() {

	tgToken := flag.String("tg", "bot123", "telegram token")
	v2raygrpc := flag.String("v2", "v2ray:8080", "v2ray grpc endpoint address")
	name := flag.String("name", "usa", "name")
	domain := flag.String("d", "tci.news", "domain")
	database_path := flag.String("db", "db.db", "database")

	flag.Parse()

	bt = tg.Bot{
		Token: *tgToken,
	}

	dbinit, err := db.GetDB(*database_path)
	datab = dbinit
	if err != nil {
		log.Fatal(err)
	}
	cc = v2rpc.GetGrpcConn(*v2raygrpc)

	//startup
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
				go HandleUpdate(up, cc, *domain, *name)
			}
		default:
			{
				qs, _ := url.ParseQuery(r.URL.RawQuery)

				cmd, ok := qs["cmd"]

				if !ok {
					var users []db.User
					datab.Find(&users, "blocked = ?", false)
					for i, usr := range users {
						usr.Quoata = v2rpc.GetUserStat(strconv.Itoa(int(usr.TgId)), cc)
						users[i] = usr
					}
					j, _ := json.Marshal(users)
					w.Write(j)
					return
				}

				switch cmd[0] {
				case "block":
					{
						tgid, ok := qs["tgid"]
						if !ok {
							w.Write([]byte("no tgid"))
							return
						}
						tgidint, _ := strconv.Atoi(tgid[0])
						user := db.User{TgId: tgidint}
						datab.First(&user)

						_, err = v2rpc.RemoveUser(tgid[0], cc)
						if err != nil {
							w.Write([]byte("failed"))
							return
						}
						user.Blocked = true
						blr, ok := qs["blr"]
						if ok {
							user.LastBlockedReason = blr[0]
						}
						datab.Save(&user)
						w.Write([]byte("blocked user"))
					}
				case "unblock":
					{
						tgid, ok := qs["tgid"]
						if !ok {
							w.Write([]byte("no tgid"))
							return
						}
						tgidint, _ := strconv.Atoi(tgid[0])
						user := db.User{TgId: tgidint}
						datab.First(&user)

						new_uuid := uuid.New()
						_, err = v2rpc.Adduser(new_uuid.String(), tgid[0], cc)

						if err != nil {
							//bt.SendMessage("failed to unblock", uidint)

							w.Write(fmt.Appendf([]byte{}, "failed to unblock %s", err))
							return
						}

						user.UUID = new_uuid.String()
						user.Blocked = false
						datab.Save(user)

						bt.SendMessage(fmt.Sprintf("new uuid generated \n\nhttps://%s/stat.html?uuid=%s&srv=%s", *domain, new_uuid.String(), *name), tgidint)
						w.Write([]byte("unblocked"))
						return
					}
				}

			}
		}

	})

	http.ListenAndServe("0.0.0.0:2020", nil)
}
