package security_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

// TestFileOwnership covers the file module's IDOR surface: a file id is a
// direct object reference, so ownership must be checked on read and delete.
func TestFileOwnership(t *testing.T) {
	w := setup(t)

	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...)
	fileID := w.alice.upload(t, "audit.png", png)

	t.Run("owner can read and download", func(t *testing.T) {
		status, _, body := w.alice.do(http.MethodGet, "/api/v1/files/"+fileID, nil, nil)
		mustStatus(t, "owner read", status, http.StatusOK, body)

		status, _, body = w.alice.do(http.MethodGet, "/api/v1/files/"+fileID+"/download", nil, nil)
		mustStatus(t, "owner download", status, http.StatusOK, body)
	})

	t.Run("another user cannot read", func(t *testing.T) {
		status, _, body := w.bob.do(http.MethodGet, "/api/v1/files/"+fileID, nil, nil)
		if isSuccess(status) {
			t.Fatalf("another user read the file (status %d)", status)
		}
		denied(t, "foreign file read", status, body)
	})

	t.Run("another user cannot download", func(t *testing.T) {
		status, _, body := w.bob.do(http.MethodGet, "/api/v1/files/"+fileID+"/download", nil, nil)
		if isSuccess(status) {
			t.Fatalf("another user downloaded the file (status %d)", status)
		}
		denied(t, "foreign file download", status, body)
	})

	t.Run("another user cannot delete", func(t *testing.T) {
		status, _, body := w.bob.do(http.MethodDelete, "/api/v1/files/"+fileID, nil, nil)
		if isSuccess(status) {
			t.Fatalf("another user deleted the file (status %d)", status)
		}
		denied(t, "foreign file delete", status, body)
	})

	t.Run("the file still exists after the failed delete", func(t *testing.T) {
		status, _, _ := w.alice.do(http.MethodGet, "/api/v1/files/"+fileID, nil, nil)
		mustStatus(t, "owner re-read", status, http.StatusOK, "")
	})

	t.Run("the list is per user", func(t *testing.T) {
		var list struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		}
		w.bob.decode("/api/v1/files", &list)
		for _, f := range list.Items {
			if f.ID == fileID {
				t.Fatal("another user's file appears in the list")
			}
		}
	})
}

// TestUploadValidation checks that the stored content type is sniffed rather
// than trusted, and that a disguised payload is refused.
func TestUploadValidation(t *testing.T) {
	w := setup(t)

	cases := []struct {
		name        string
		filename    string
		content     []byte
		contentType string
		wantOK      bool
	}{
		{"real png", "ok.png", append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...), "image/png", true},
		{"html disguised as png", "evil.png", []byte("<html><script>alert(1)</script></html>"), "image/png", false},
		{"svg", "image.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), "image/svg+xml", false},
		{"path traversal in name", "../../etc/passwd.png", append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...), "image/png", true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			status, data, body := w.alice.uploadRaw(tc.filename, tc.content, tc.contentType)
			if tc.wantOK && !isSuccess(status) {
				t.Fatalf("status %d, want success (body: %s)", status, truncate(body))
			}
			if !tc.wantOK && isSuccess(status) {
				t.Fatalf("a %s upload was accepted", tc.name)
			}
			if !isSuccess(status) {
				return
			}

			var file struct {
				ID           string `json:"id"`
				OriginalName string `json:"original_name"`
			}
			decodeInto(t, data, &file)

			// A traversal name must not survive into the visible metadata.
			if strings.Contains(file.OriginalName, "..") || strings.Contains(file.OriginalName, "/") {
				t.Errorf("original_name = %q keeps path separators", file.OriginalName)
			}

			// The download must not be served as executable markup.
			res := w.alice.download(t, file.ID)
			if ct := res.Header.Get("Content-Type"); strings.Contains(ct, "html") || strings.Contains(ct, "svg") {
				t.Errorf("download Content-Type = %q", ct)
			}
			if cd := res.Header.Get("Content-Disposition"); cd == "" || !strings.Contains(cd, "attachment") {
				t.Errorf("Content-Disposition = %q, want an attachment", cd)
			}
			if nosniff := res.Header.Get("X-Content-Type-Options"); nosniff != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q on download", nosniff)
			}
		})
	}
}

// ------------------------------------------------------------------- helpers

func (c *client) upload(t *testing.T, filename string, content []byte) string {
	t.Helper()
	status, data, body := c.uploadRaw(filename, content, "image/png")
	if status != http.StatusCreated {
		t.Fatalf("upload: status %d (body: %s)", status, truncate(body))
	}
	var file struct {
		ID string `json:"id"`
	}
	decodeInto(t, data, &file)
	return file.ID
}

func (c *client) uploadRaw(filename string, content []byte, contentType string) (int, []byte, string) {
	c.t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		c.t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		c.t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		c.t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, c.base+"/api/v1/files", &buf)
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatal(err)
	}

	var env envelope
	_ = json.Unmarshal(body, &env)
	_ = contentType
	return res.StatusCode, env.Data, string(body)
}

func (c *client) download(t *testing.T, id string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, c.base+"/api/v1/files/"+id+"/download", nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}
