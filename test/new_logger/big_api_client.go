package new_logger

type Message string

type BigAPIClient struct {
	// пропустим код
}

func (c *BigAPIClient) Connect() error {
	return ErrFoundNum
}

func (c *BigAPIClient) Close() error {
	return ErrFoundNum
}

func (c *BigAPIClient) FetchMessages() ([]Message, error) {
	return []Message{}, ErrFoundNum
}

func (c *BigAPIClient) SendMessage(email string, message string) error {
	return ErrFoundNum
}

func (c *BigAPIClient) SendStatus(status string) error {
	return ErrFoundNum
}
