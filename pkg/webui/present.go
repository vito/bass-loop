package webui

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"html/template"
	"math/rand"
	"time"

	svg "github.com/ajstarks/svgo"
	"github.com/vito/bass-loop/html"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/invaders"
)

type VertexTemplateContext struct {
	Num int

	*models.Vertex

	Duration string
	Lines    []Line
}

type Line struct {
	Number  int
	Content template.HTML
}

var tmpl = template.Must(template.ParseFS(html.FS, "*.tmpl"))

func duration(dt time.Duration) string {
	prec := 1
	sec := dt.Seconds()
	if sec < 10 {
		prec = 2
	} else if sec < 100 {
		prec = 1
	}

	return fmt.Sprintf("%.[2]*[1]fs", sec, prec)
}

func thunkAvatar(thunkDigest string) (template.HTML, error) {
	h := fnv.New64a()
	if _, err := h.Write([]byte(thunkDigest)); err != nil {
		return "", err
	}

	invader := &invaders.Invader{}
	invader.Set(rand.New(rand.NewSource(int64(h.Sum64()))))

	avatarSvg := new(bytes.Buffer)
	canvas := svg.New(avatarSvg)

	cellSize := 9
	canvas.Startview(
		cellSize*invaders.Width,
		cellSize*invaders.Height,
		0,
		0,
		cellSize*invaders.Width,
		cellSize*invaders.Height,
	)
	canvas.Group()

	for row := range invader {
		y := row * cellSize

		for col := range invader[row] {
			x := col * cellSize
			shade := invader[row][col]

			var color string
			switch shade {
			case invaders.Background:
				color = "transparent"
			case invaders.Shade1:
				color = "var(--base08)"
			case invaders.Shade2:
				color = "var(--base09)"
			case invaders.Shade3:
				color = "var(--base0A)"
			case invaders.Shade4:
				color = "var(--base0B)"
			case invaders.Shade5:
				color = "var(--base0C)"
			case invaders.Shade6:
				color = "var(--base0D)"
			case invaders.Shade7:
				color = "var(--base0E)"
			default:
				return "", fmt.Errorf("invalid shade: %v", shade)
			}

			canvas.Rect(
				x, y,
				cellSize, cellSize,
				fmt.Sprintf("fill: %s", color),
			)
		}
	}

	canvas.Gend()
	canvas.End()

	return template.HTML(avatarSvg.String()), nil
}
