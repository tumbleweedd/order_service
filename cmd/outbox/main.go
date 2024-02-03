package outbox

import "time"

func main() {
	ticker := time.NewTicker(time.Minute * 1)

	for {
		select {
		case <-ticker.C:

		}
	}
}
