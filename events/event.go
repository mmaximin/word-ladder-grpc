package events

const QueueName = "game_events"

type GameEvent struct {
	Username string `json:"username"`
	GameID   int32  `json:"game_id"`
	Result   string `json:"result"`
	Turns    int32  `json:"turns"`
}
