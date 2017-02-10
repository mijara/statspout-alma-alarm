package notifier

import (
	"io"
	"net/smtp"

	"github.com/alecthomas/template"
	"github.com/mijara/statspout/log"
)

// Email sends via email all messages, using the configured SMTP sender, sender and recipients.
func Email(messages []string) {
	cli, err := smtp.Dial("mail.google.com:25")
	if err != nil {
		log.Error.Println("send email notification failed: " + err.Error())
		return
	}
	defer cli.Close()

	cli.Mail("sender@example.com")
	cli.Rcpt("recipient@example.com")

	wc, err := cli.Data()
	if err != nil {
		log.Error.Println("send email notification failed: " + err.Error())
		return
	}
	defer wc.Close()

	err = renderTemplate(messages, wc)
	if err != nil {
		log.Error.Println("send email notification failed: " + err.Error())
		return
	}
}

// renderTemplate renders all messages to the io.Writer of the email, using a fixed template.
func renderTemplate(messages []string, wc io.Writer) error {
	tp, err := template.New("email").Parse(
		"{{range .}}{{.}}\n{{end}}",
	)
	if err != nil {
		return err
	}

	err = tp.Execute(wc, messages)
	if err != nil {
		return err
	}

	return nil
}
