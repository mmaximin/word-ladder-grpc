package events

// QueueName is the RabbitMQ queue used by the game server (publisher)
// and the stats server (consumer).
const QueueName = "game_events"

// GameEvent is the JSON message sent when a game finishes.
type GameEvent struct {
	Username string `json:"username"`
	GameID   int32  `json:"game_id"`
	Result   string `json:"result"` // "WIN" or "LOSS"
	Turns    int32  `json:"turns"`
}
