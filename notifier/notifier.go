package notifier

// Notifier represent an entity that is able to notify using some engine.
type Notifier interface {
	Notify(messages []string)
	Close()
}
