package render

import (
	"bytes"
	"html/template"

	terminal "github.com/buildkite/terminal-to-html/v3"
	"github.com/dmitriy/curlstreet/assets"
	"github.com/dmitriy/curlstreet/internal/quote"
)

var htmlTemplate = template.Must(template.New("base").Parse(string(assets.BaseHTML)))

type htmlData struct {
	Title string
	Body  template.HTML
}

func RenderHTML(quotes []quote.QuoteResult) (string, error) {
	ansi, err := RenderText(quotes)
	if err != nil {
		return "", err
	}

	rendered := terminal.Render([]byte(ansi))

	title := "ticker"
	if len(quotes) > 0 && quotes[0].Quote != nil {
		q := quotes[0].Quote
		title = q.Symbol + " — " + formatPrice(q)
	}

	var buf bytes.Buffer
	if err := htmlTemplate.Execute(&buf, htmlData{
		Title: title,
		Body:  template.HTML(rendered),
	}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func RenderError(code int, msg string, format quote.ResponseFormat) string {
	switch format {
	case quote.ResponseFormatJSON:
		return `{"error":"` + msg + `"}`
	case quote.ResponseFormatHTML:
		escaped := template.HTMLEscapeString(msg)
		var buf bytes.Buffer
		htmlTemplate.Execute(&buf, htmlData{
			Title: "Error",
			Body:  template.HTML("<span style=\"color:red\">" + escaped + "</span>"),
		})
		return buf.String()
	default:
		return msg
	}
}
