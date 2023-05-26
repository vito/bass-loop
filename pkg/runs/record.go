package runs

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"time"

	"github.com/aoldershaw/ansi"
	"github.com/vito/bass-loop/pkg/blobs"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass/pkg/zapctx"
	"github.com/vito/progrock"
	"github.com/vito/progrock/ui"
	"go.uber.org/zap"
)

func Record(ctx context.Context, db models.DB, bucket *blobs.Bucket, run *models.Run, tape *progrock.Tape, ok bool) error {
	logger := zapctx.FromContext(ctx)

	completedAt := models.NewTime(time.Now().UTC())
	run.EndTime = &completedAt

	if ok {
		run.Succeeded = sql.NullInt64{Int64: 1, Valid: true}
	} else if ctx.Err() != nil {
		run.Succeeded = sql.NullInt64{Int64: 0, Valid: true}
	} else {
		run.Succeeded = sql.NullInt64{Int64: 0, Valid: true}
	}

	err := tape.EachVertex(func(v *progrock.Vertex, l *ui.Vterm) error {
		var startTime, endTime models.Time
		if v.Started != nil {
			startTime = models.NewTime(v.Started.AsTime().UTC())
		}
		if v.Completed != nil {
			endTime = models.NewTime(v.Completed.AsTime().UTC())
		}

		var vErr sql.NullString
		if v.Error != nil {
			vErr.String = v.GetError()
			vErr.Valid = true
		}

		var cached int
		if v.Cached {
			cached = 1
		}

		vtx := &models.Vertex{
			Digest:    v.Id,
			RunID:     run.ID,
			Name:      v.Name,
			StartTime: &startTime,
			EndTime:   &endTime,
			Error:     vErr,
			Cached:    cached,
		}

		htmlBuf := new(bytes.Buffer)
		if l.UsedHeight() > 0 {
			logs := l.Bytes(0, l.UsedHeight())

			if err := bucket.WriteAll(ctx, blobs.VertexRawLogKey(vtx), logs, nil); err != nil {
				return fmt.Errorf("store raw logs: %w", err)
			}

			var lines ansi.Lines
			writer := ansi.NewWriter(&lines,
				// arbitrary, matched my screen
				ansi.WithInitialScreenSize(67, 316))
			if _, err := writer.Write(logs); err != nil {
				return fmt.Errorf("write log: %w", err)
			}

			if err := ANSIHTML.Execute(htmlBuf, lines); err != nil {
				return fmt.Errorf("render html: %w", err)
			}

			if err := bucket.WriteAll(ctx, blobs.VertexHTMLLogKey(vtx), htmlBuf.Bytes(), nil); err != nil {
				return fmt.Errorf("store html logs: %w", err)
			}
		}

		for {
			if err := vtx.Save(ctx, db); err != nil {
				// TODO why is this happening so often even with retrying?
				logger.Error("failed to save vertex", zap.Error(err))
				time.Sleep(time.Second)
				continue
			}

			break
		}

		for _, input := range v.Inputs {
			edge := models.VertexEdge{
				SourceDigest: input,
				TargetDigest: v.Id,
			}

			_, err := models.VertexEdgeBySourceDigestTargetDigest(ctx, db, edge.SourceDigest, edge.TargetDigest)
			if err != nil && errors.Is(err, sql.ErrNoRows) {
				// this could conflict with another edge, but that's ok; we just do
				// the above check to make the logs less noisy
				if err := edge.Insert(ctx, db); err != nil {
					logger.Warn("insert edge", zap.Error(err))
				}
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("store vertex logs: %w", err)
	}

	err = run.Update(ctx, db)
	if err != nil {
		return fmt.Errorf("update thunk run: %w", err)
	}

	return nil
}

// TODO: support modifiers (bold/etc) - it's a bit tricky, may need changes
// upstream
var ANSIHTML = template.Must(template.New("ansi").Parse(`{{- range . -}}
	<span class="ansi-line">
		{{- range . -}}
		{{- if or .Style.Foreground .Style.Background .Style.Modifier -}}
			<span class="{{with .Style.Foreground}}fg-{{.}}{{end}}{{with .Style.Background}} bg-{{.}}{{end}}">
				{{- printf "%s" .Data -}}
			</span>
		{{- else -}}
			{{- printf "%s" .Data -}}
		{{- end -}}
		{{- end -}}
	</span>
{{end}}`))
