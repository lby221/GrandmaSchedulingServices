package distributor

import (
	"strings"
	"ws"
)

func sendWebSocketMsg(endpoint string, message string) error {
	ep := strings.Split(endpoint, ".")

	if len(ep) < 2 {
		return ErrorInvalidEndpointOrBody
	}
	id := ep[0]
	key := ep[1]
	return ws.Send(id, key, message)
}
