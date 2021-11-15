package docit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	_ "embed" //embed: read file

	as "github.com/godevsig/adaptiveservice"
)

//go:embed github-markdown.css
var css string

// Request is the message sent by client.
type Request struct {
	Text string `json:"text"`
}

// Response is the message replied by server.
type Response struct {
	HTML string
}

const (
	layout = `
	<style>
		.markdown-body {
			box-sizing: border-box;
			min-width: 200px;
			max-width: 980px;
			margin: 0 auto;
			padding: 45px;
		}
		@media (max-width: 767px) {
			.markdown-body {
				padding: 15px;
			}
		}
	</style>
	`
)

/*
curl \
	-X POST \
	-H "Accept: application/vnd.github.v3+json" \
	https://api.github.com/markdown \
	-d '{"text":"text"}'
*/

// Handle handles msg.
func (msg *Request) Handle(stream as.ContextStream) (reply interface{}) {
	payload, _ := json.Marshal(msg)
	body := bytes.NewReader(payload)

	req, err := http.NewRequest("POST", "https://api.github.com/markdown", body)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return &Response{HTML: genHTML(string(respBody))}
}

func genHTML(body string) string {
	return fmt.Sprintf("<style>%s</style>%s<article class=\"markdown-body\">%s</article>", css, layout, body)
}

//Run start service.
func Run(args []string) (err error) {
	var opts = []as.Option{as.WithScope(as.ScopeWAN)}
	server := as.NewServer(opts...).SetPublisher("platform")
	if err := server.Publish("markdown",
		knownMsgs,
	); err != nil {
		fmt.Printf("create markdown server failed: %v", err)
		return err
	}
	err = server.Serve()
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

var knownMsgs = []as.KnownMessage{
	(*Request)(nil),
}

func init() {
	as.RegisterType((*Request)(nil))
	as.RegisterType((*Response)(nil))
}
