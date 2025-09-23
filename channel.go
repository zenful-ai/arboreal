package arboreal

import (
	"bufio"
	"fmt"
	"github.com/ajg/form"
	"github.com/twilio/twilio-go"
	"github.com/twilio/twilio-go/client"
	api "github.com/twilio/twilio-go/rest/api/v2010"
	"net/http"
	"os"
	"strings"
)

type ChannelMessage struct {
	Id      string `json:"id"`
	Content string `json:"content"`
}

type Channel interface {
	AllocateID() string
	Send(*ChannelMessage) error
	Receive() (*ChannelMessage, error)
}

type TerminalChannel struct{}

func (TerminalChannel) AllocateID() string {
	return ""
}

func (TerminalChannel) Send(m *ChannelMessage) error {
	fmt.Print("[Assistant Response]\n\n")
	fmt.Print(m.Content)
	fmt.Print("\n\n")

	return nil
}

func (TerminalChannel) Receive() (*ChannelMessage, error) {
	fmt.Print("[User Message]\n\n")

	scn := bufio.NewScanner(os.Stdin)
	var lines []string
	for scn.Scan() {
		line := scn.Text()
		if len(line) == 1 {
			// Group Separator (GS ^]): ctrl-]
			if line[0] == '$' || line[0] == '\x1D' {
				break
			}
		}
		lines = append(lines, line)
	}
	fmt.Println()

	return &ChannelMessage{
		Id:      "",
		Content: strings.Join(lines, "\n"),
	}, nil
}

type TwilioSMSChannel struct {
	AllowList []string
	inbound   chan ChannelMessage
}

func CreateTwilioSMSChannel() *TwilioSMSChannel {
	var c TwilioSMSChannel

	c.inbound = make(chan ChannelMessage, 300)

	go func() {
		authToken := os.Getenv("TWILIO_AUTH_TOKEN")
		requestValidator := client.NewRequestValidator(authToken)

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Hello, world!")
		})
		http.HandleFunc("/api/v1/twilio", func(w http.ResponseWriter, r *http.Request) {
			params := make(map[string]string)
			err := form.NewDecoder(r.Body).Decode(&params)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			requestURL := fmt.Sprintf("https://%s%s", r.Host, r.RequestURI)

			signature := r.Header.Get("X-Twilio-Signature")

			fmt.Println("Got twilio message!")
			fmt.Printf("Validating against %s, signature: %s\n", requestURL, signature)

			if !requestValidator.Validate(requestURL, params, signature) {
				fmt.Println("Twilio message did not validate")
				w.WriteHeader(http.StatusForbidden)
				return
			}

			var msg ChannelMessage

			// TODO: Gate on proper user?
			if id, ok := params["From"]; ok {
				if len(c.AllowList) > 0 {
					var found bool
					for _, allowed := range c.AllowList {
						if allowed == id {
							found = true
							break
						}
					}

					if !found {
						fmt.Printf("Twilio message from unknown user: %s\n", id)
						w.WriteHeader(http.StatusOK)
						return
					}
				}

				msg.Id = id
			} else {
				fmt.Println("No From parameter")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if body, ok := params["Body"]; ok {
				msg.Content = body
			} else {
				fmt.Println("No Body parameter")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			c.inbound <- msg
			w.WriteHeader(http.StatusOK)
		})

		http.ListenAndServe(":8000", nil)
	}()

	return &c
}

func (c *TwilioSMSChannel) AllocateID() string {
	return ""
}

func (c *TwilioSMSChannel) Send(m *ChannelMessage) error {
	tc := twilio.NewRestClient()

	params := &api.CreateMessageParams{}
	params.SetBody(m.Content)
	params.SetMessagingServiceSid(os.Getenv("TWILIO_MESSAGING_SID"))
	params.SetTo(m.Id)

	_, err := tc.Api.CreateMessage(params)
	if err != nil {
		return err
	}

	return nil
}

func (c *TwilioSMSChannel) Receive() (*ChannelMessage, error) {
	m := <-c.inbound
	return &m, nil
}
