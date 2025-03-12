package repositories

type UserRef struct {
	ID       string `json:"id" bson:"id"`
	Nickname string `json:"nickname" bson:"nickname"`
}
