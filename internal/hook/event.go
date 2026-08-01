package hook

import "encoding/json"

type Event struct {
	HookEventName string `json:"hook_event_name"`
	Prompt        string `json:"prompt"`
}

func Parse(data []byte) (Event, error) {
	var e Event
	err := json.Unmarshal(data, &e)
	return e, err
}
