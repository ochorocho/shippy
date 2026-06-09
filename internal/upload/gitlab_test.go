package upload

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newTestUploader builds a GitLabUploader pointed at a test server. The struct
// fields are unexported, but this test lives in package upload, so it can set
// baseURL to the httptest server (http://...) without touching the production
// constructor, which intentionally forces https://.
func newTestUploader(serverURL, projectPath, token string, client *http.Client) *GitLabUploader {
	return &GitLabUploader{
		baseURL:     serverURL,
		projectPath: projectPath,
		token:       token,
		client:      client,
	}
}

// writeTempFile creates a file with the given content and returns its path.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

func TestGitLabUploader_Upload_Success(t *testing.T) {
	const (
		projectPath = "namespace/project"
		token       = "glpat-secret-token"
		fileContent = "this is a backup archive payload"
	)

	var (
		gotMethod      string
		gotEscapedPath string
		gotToken       string
		gotBody        []byte
		gotLength      int64
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotEscapedPath = r.URL.EscapedPath()
		gotToken = r.Header.Get("PRIVATE-TOKEN")
		gotLength = r.ContentLength
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"message":"201 Created"}`))
	}))
	defer server.Close()

	filePath := writeTempFile(t, "backup-production-20240109T120000.zip", fileContent)
	g := newTestUploader(server.URL, projectPath, token, server.Client())

	result, err := g.Upload(filePath, UploadOptions{
		PackageName:    "my-app",
		PackageVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("Upload() unexpected error: %v", err)
	}

	// The GitLab Generic Packages API expects a PUT to a very specific path.
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}

	// Project path must be URL-encoded (the "/" becomes "%2F"), the rest of the
	// segments path-escaped. This is the contract that makes the upload land in
	// the right package registry.
	wantPath := "/api/v4/projects/namespace%2Fproject/packages/generic/my-app/1.0.0/backup-production-20240109T120000.zip"
	if gotEscapedPath != wantPath {
		t.Errorf("request path = %q, want %q", gotEscapedPath, wantPath)
	}

	if gotToken != token {
		t.Errorf("PRIVATE-TOKEN header = %q, want %q", gotToken, token)
	}

	if string(gotBody) != fileContent {
		t.Errorf("request body = %q, want %q", string(gotBody), fileContent)
	}

	if gotLength != int64(len(fileContent)) {
		t.Errorf("Content-Length = %d, want %d", gotLength, len(fileContent))
	}

	// The result should report the uploaded filename and the package browse URL.
	if !strings.Contains(result.Message, "backup-production-20240109T120000.zip") {
		t.Errorf("result message = %q, want it to mention the filename", result.Message)
	}
	wantURL := server.URL + "/namespace/project/-/packages"
	if result.URL != wantURL {
		t.Errorf("result URL = %q, want %q", result.URL, wantURL)
	}
}

func TestGitLabUploader_Upload_PathEncoding(t *testing.T) {
	tests := []struct {
		name        string
		projectPath string
		packageName string
		version     string
		fileName    string
		wantPath    string
	}{
		{
			name:        "subgroup project path is fully escaped",
			projectPath: "group/subgroup/project",
			packageName: "backups",
			version:     "20240109T120000",
			fileName:    "backup.zip",
			wantPath:    "/api/v4/projects/group%2Fsubgroup%2Fproject/packages/generic/backups/20240109T120000/backup.zip",
		},
		{
			name:        "spaces in package name are escaped",
			projectPath: "ns/proj",
			packageName: "my app",
			version:     "1.0.0",
			fileName:    "dump.sql.gz",
			wantPath:    "/api/v4/projects/ns%2Fproj/packages/generic/my%20app/1.0.0/dump.sql.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.EscapedPath()
				w.WriteHeader(http.StatusCreated)
			}))
			defer server.Close()

			filePath := writeTempFile(t, tt.fileName, "payload")
			g := newTestUploader(server.URL, tt.projectPath, "token", server.Client())

			_, err := g.Upload(filePath, UploadOptions{
				PackageName:    tt.packageName,
				PackageVersion: tt.version,
			})
			if err != nil {
				t.Fatalf("Upload() unexpected error: %v", err)
			}

			if gotPath != tt.wantPath {
				t.Errorf("request path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

func TestGitLabUploader_Upload_ServerError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantInErr  []string
	}{
		{
			name:       "unauthorized token",
			statusCode: http.StatusUnauthorized,
			body:       `{"message":"401 Unauthorized"}`,
			wantInErr:  []string{"HTTP 401", "Unauthorized"},
		},
		{
			name:       "server error includes response body",
			statusCode: http.StatusInternalServerError,
			body:       "boom",
			wantInErr:  []string{"HTTP 500", "boom"},
		},
		{
			name:       "200 OK is treated as failure (API must return 201)",
			statusCode: http.StatusOK,
			body:       "ok but wrong",
			wantInErr:  []string{"HTTP 200"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			filePath := writeTempFile(t, "backup.zip", "payload")
			g := newTestUploader(server.URL, "ns/proj", "token", server.Client())

			result, err := g.Upload(filePath, UploadOptions{
				PackageName:    "app",
				PackageVersion: "1.0.0",
			})
			if err == nil {
				t.Fatalf("Upload() expected error for status %d, got nil (result: %+v)", tt.statusCode, result)
			}
			for _, want := range tt.wantInErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q, want it to contain %q", err.Error(), want)
				}
			}
		})
	}
}

func TestGitLabUploader_Upload_FileNotFound(t *testing.T) {
	// The server should never be hit; record if it is.
	hit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	g := newTestUploader(server.URL, "ns/proj", "token", server.Client())

	_, err := g.Upload(filepath.Join(t.TempDir(), "does-not-exist.zip"), UploadOptions{
		PackageName:    "app",
		PackageVersion: "1.0.0",
	})
	if err == nil {
		t.Fatal("Upload() expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to open file") {
		t.Errorf("error %q, want it to mention opening the file", err.Error())
	}
	if hit {
		t.Error("server should not be contacted when the file cannot be opened")
	}
}

// TestGitLabUploader_Upload_EmptyFile guards the edge case of a zero-byte
// upload: the request must still reach the server as a PUT and succeed.
func TestGitLabUploader_Upload_EmptyFile(t *testing.T) {
	var (
		gotMethod string
		gotBody   []byte
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	filePath := writeTempFile(t, "empty.zip", "")
	g := newTestUploader(server.URL, "ns/proj", "token", server.Client())

	if _, err := g.Upload(filePath, UploadOptions{PackageName: "app", PackageVersion: "1.0.0"}); err != nil {
		t.Fatalf("Upload() unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if len(gotBody) != 0 {
		t.Errorf("body = %q, want empty", string(gotBody))
	}
}

// TestNewGitLabUploader_ForcesHTTPS documents the security-relevant default:
// the production constructor always builds an https:// base URL from the host.
func TestNewGitLabUploader_ForcesHTTPS(t *testing.T) {
	g := NewGitLabUploader("gitlab.example.com", "ns/proj", "token")
	if !strings.HasPrefix(g.baseURL, "https://") {
		t.Errorf("baseURL = %q, want it to start with https://", g.baseURL)
	}
	if g.client.Timeout != 30*time.Minute {
		t.Errorf("client timeout = %v, want 30m", g.client.Timeout)
	}
}
