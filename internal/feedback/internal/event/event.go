package event

type CreditsEvent struct {
	Uid int64 `json:"uid"`
}

func (CreditsEvent) Topic() string {
	return "feedback_credits_events"
}
