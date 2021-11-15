package docit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	_ "embed" //embed: read file

	as "github.com/godevsig/adaptiveservice"
)

//go:embed github-markdown.css
var css string

// MarkdownRequest is the message sent by client.
// Reply HTMLResponse.
type MarkdownRequest struct {
	Text string `json:"text"`
}

// HTMLResponse is the message replied by server.
type HTMLResponse struct {
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
func (msg *MarkdownRequest) Handle(stream as.ContextStream) (reply interface{}) {
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return &HTMLResponse{HTML: fmt.Sprintf("<style>%s</style>%s<article class=\"markdown-body\">%s</article>", css, layout, string(respBody))}
}

var knownMsgs = []as.KnownMessage{
	(*MarkdownRequest)(nil),
}

func init() {
	as.RegisterType((*MarkdownRequest)(nil))
	as.RegisterType((*HTMLResponse)(nil))
}
