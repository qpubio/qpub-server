package worker

import "encoding/json"

// ParseQueues decodes the stored queues JSON array.
func ParseQueues(raw string) []string {
	if raw == "" {
		return nil
	}
	var queues []string
	if err := json.Unmarshal([]byte(raw), &queues); err != nil {
		return nil
	}
	return queues
}

// RemoveQueue returns updated queues JSON with queueName removed.
func RemoveQueue(raw, queueName string) string {
	queues := ParseQueues(raw)
	if len(queues) == 0 {
		return raw
	}
	out := make([]string, 0, len(queues))
	for _, q := range queues {
		if q != queueName {
			out = append(out, q)
		}
	}
	data, err := json.Marshal(out)
	if err != nil {
		return raw
	}
	return string(data)
}
