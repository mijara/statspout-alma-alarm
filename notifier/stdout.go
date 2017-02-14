package notifier

import "github.com/mijara/statspout/log"

// Stdout is the basic notifier that outputs alarm to Standard output.
type Stdout struct {
}

// NewStdout creates a new instance of the Stdout notifier.
func NewStdout() *Stdout {
	return &Stdout{}
}

// Notify will print each message.
func (s *Stdout) Notify(messages []string) {
	for _, message := range messages {
		log.Warning.Println(message)
	}
}

// Close will do nothing.
func (s *Stdout) Close() {
}
