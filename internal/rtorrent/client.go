package rtorrent

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/swamp2k/copyarr/internal/config"
)

type Torrent struct {
	Hash, Name, BasePath string
	Complete             bool
}

type Client struct {
	cfg  config.RTorrent
	http *http.Client
}

func New(cfg config.RTorrent) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: 15 * time.Second}}
}

type xValue struct {
	String *string `xml:"string"`
	Int    *int64  `xml:"int"`
	I4     *int64  `xml:"i4"`
	Array  *xArray `xml:"array"`
}
type xArray struct {
	Values []xValue `xml:"data>value"`
}
type response struct {
	Params []struct {
		Value xValue `xml:"value"`
	} `xml:"params>param"`
	Fault *struct {
		Value xValue `xml:"value"`
	} `xml:"fault"`
}

func esc(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func (c *Client) Torrents(ctx context.Context) ([]Torrent, error) {
	view := c.cfg.View
	if view == "" {
		view = "main"
	}
	body := `<?xml version="1.0"?><methodCall><methodName>d.multicall2</methodName><params>` +
		`<param><value><string></string></value></param>` +
		`<param><value><string>` + esc(view) + `</string></value></param>` +
		`<param><value><string>d.hash=</string></value></param>` +
		`<param><value><string>d.name=</string></value></param>` +
		`<param><value><string>d.complete=</string></value></param>` +
		`<param><value><string>d.base_path=</string></value></param>` +
		`</params></methodCall>`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml")
	if c.cfg.Username != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf("rtorrent HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	var rr response
	if err := xml.NewDecoder(res.Body).Decode(&rr); err != nil {
		return nil, err
	}
	if len(rr.Params) == 0 || rr.Params[0].Value.Array == nil {
		return nil, fmt.Errorf("unexpected rtorrent XML-RPC response")
	}
	var out []Torrent
	for _, rowv := range rr.Params[0].Value.Array.Values {
		if rowv.Array == nil || len(rowv.Array.Values) < 4 {
			continue
		}
		row := rowv.Array.Values
		t := Torrent{Hash: str(row[0]), Name: str(row[1]), Complete: num(row[2]) != 0, BasePath: str(row[3])}
		out = append(out, t)
	}
	return out, nil
}

func str(v xValue) string {
	if v.String != nil {
		return *v.String
	}
	return ""
}

func num(v xValue) int64 {
	if v.Int != nil {
		return *v.Int
	}
	if v.I4 != nil {
		return *v.I4
	}
	return 0
}


func (c *Client) Completed(ctx context.Context) ([]Torrent, error) {
	all, err := c.Torrents(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Torrent, 0, len(all))
	for _, t := range all {
		if t.Complete {
			out = append(out, t)
		}
	}
	return out, nil
}
