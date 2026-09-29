package backend

type Message string

// BigAPIClient stands in for a client whose network implementation is omitted.
type BigAPIClient struct{}

func (*BigAPIClient) Connect() error { return nil }
func (*BigAPIClient) Close() error   { return nil }
func (*BigAPIClient) FetchMessages() ([]Message, error) {
	return nil, nil
}
func (*BigAPIClient) SendMessage(email, message string) error { return nil }
func (*BigAPIClient) SendStatus(status string) error          { return nil }

type MessageSender interface {
	SendMessage(email, message string) error
}

func SendWelcome(sender MessageSender, email string) error {
	return sender.SendMessage(email, "Welcome")
}
