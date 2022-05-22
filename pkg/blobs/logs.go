package blobs

import (
	"path"

	"github.com/vito/bass-loop/pkg/models"
)

func VertexRawLogKey(vtx *models.Vertex) string {
	return path.Join("logs", vtx.RunID, vtx.Digest)
}

func VertexHTMLLogKey(vtx *models.Vertex) string {
	return path.Join("logs", vtx.RunID, vtx.Digest+".html")
}
