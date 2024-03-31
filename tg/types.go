package tg

type Update struct {
	Updateid int     `json:"update_id"`
	Message  Message `json:"message,omitempty"`
}

type Message struct {
	Messageid int    `json:"message_id"`
	From      User   `json:"from,omitempty"`
	Text      string `json:"text,omitempty"`
}

type User struct {
	Id     int  `json:"id"`
	Isprem bool `json:"is_premium,omitempty"`
}
