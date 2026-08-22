package bedrock

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// Attachment decoding for the Converse path. AIMessage carries Images and
// Files as data URLs or fetchable URLs; Converse wants raw bytes with a
// declared format, so each reference is resolved here at request time.

var imageFormats = map[string]types.ImageFormat{
	"png":  types.ImageFormatPng,
	"jpg":  types.ImageFormatJpeg,
	"jpeg": types.ImageFormatJpeg,
	"gif":  types.ImageFormatGif,
	"webp": types.ImageFormatWebp,
}

var documentFormats = map[string]types.DocumentFormat{
	"pdf":  types.DocumentFormatPdf,
	"csv":  types.DocumentFormatCsv,
	"doc":  types.DocumentFormatDoc,
	"docx": types.DocumentFormatDocx,
	"xls":  types.DocumentFormatXls,
	"xlsx": types.DocumentFormatXlsx,
	"html": types.DocumentFormatHtml,
	"txt":  types.DocumentFormatTxt,
	"md":   types.DocumentFormatMd,
}

var mediaClient = &http.Client{Timeout: 30 * time.Second}

// fetchRef resolves one attachment reference to raw bytes plus a lowercase
// extension hint. Data URLs decode locally; http(s) URLs are fetched.
func fetchRef(ref string) (data []byte, ext string, err error) {
	if strings.HasPrefix(ref, "data:") {
		meta, payload, ok := strings.Cut(ref[len("data:"):], ",")
		if !ok {
			return nil, "", fmt.Errorf("malformed data URL")
		}
		mime, _, _ := strings.Cut(meta, ";")
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, "", fmt.Errorf("data URL payload: %w", err)
		}
		_, sub, _ := strings.Cut(mime, "/")
		return data, strings.ToLower(sub), nil
	}

	resp, err := mediaClient.Get(ref)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fetching attachment: %d", resp.StatusCode)
	}
	data, err = io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, "", err
	}
	// Prefer the served content type; fall back to the URL path's extension.
	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "/") {
		_, sub, _ := strings.Cut(ct, "/")
		sub, _, _ = strings.Cut(sub, ";")
		return data, strings.ToLower(strings.TrimSpace(sub)), nil
	}
	u := ref
	if q := strings.IndexByte(u, '?'); q != -1 {
		u = u[:q]
	}
	return data, strings.ToLower(strings.TrimPrefix(path.Ext(u), ".")), nil
}

func normalizeExt(ext string) string {
	switch ext {
	case "jpg":
		return "jpeg"
	case "plain", "text":
		return "txt"
	case "markdown":
		return "md"
	case "vnd.openxmlformats-officedocument.wordprocessingml.document":
		return "docx"
	case "vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return "xlsx"
	case "msword":
		return "doc"
	case "vnd.ms-excel":
		return "xls"
	}
	return ext
}

// imageBlock resolves one image reference into a Converse content block.
func imageBlock(ref string) (types.ContentBlock, error) {
	data, ext, err := fetchRef(ref)
	if err != nil {
		return nil, err
	}
	format, ok := imageFormats[normalizeExt(ext)]
	if !ok {
		return nil, fmt.Errorf("unsupported image format %q", ext)
	}
	return &types.ContentBlockMemberImage{Value: types.ImageBlock{
		Format: format,
		Source: &types.ImageSourceMemberBytes{Value: data},
	}}, nil
}

// documentBlock resolves one file reference into a Converse document block.
// name appears to the model; Converse requires it non-empty.
func documentBlock(ref, name string) (types.ContentBlock, error) {
	data, ext, err := fetchRef(ref)
	if err != nil {
		return nil, err
	}
	format, ok := documentFormats[normalizeExt(ext)]
	if !ok {
		return nil, fmt.Errorf("unsupported document format %q", ext)
	}
	if name == "" {
		name = "attachment"
	}
	return &types.ContentBlockMemberDocument{Value: types.DocumentBlock{
		Format: format,
		Name:   &name,
		Source: &types.DocumentSourceMemberBytes{Value: data},
	}}, nil
}
