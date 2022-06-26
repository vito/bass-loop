package present

import (
	"bytes"
	"fmt"

	chtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
)

func RenderJSON(payload []byte) (string, error) {
	lexer := lexers.Get("json")
	if lexer == nil {
		lexer = lexers.Fallback
	}

	iterator, err := lexer.Tokenise(nil, string(payload))
	if err != nil {
		return "", fmt.Errorf("tokenise: %w", err)
	}

	formatter := chtml.New(
		chtml.PreventSurroundingPre(false),
		chtml.WithClasses(true),
	)

	hlJSON := new(bytes.Buffer)
	err = formatter.Format(hlJSON, styles.Fallback, iterator)
	if err != nil {
		return "", fmt.Errorf("format json: %w", err)
	}

	return hlJSON.String(), nil
}
