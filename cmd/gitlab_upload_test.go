package cmd

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
)

// stubTransport records uploads instead of sending them.
type stubTransport struct {
	status   int
	requests []*http.Request
	bodies   []string
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body.Close()
	s.requests = append(s.requests, req)
	s.bodies = append(s.bodies, string(body))
	return &http.Response{
		StatusCode: s.status,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("stub response")),
		Request:    req,
	}, nil
}

// runUpload runs `shippy gitlab:upload backup.zip <args>` in a temporary git
// repository whose origin is gitlab.example.com/acme/site.
func runUpload(t *testing.T, status int, args ...string) (*stubTransport, error) {
	t.Helper()

	dir := t.TempDir()
	t.Chdir(dir)
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateRemote(&gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://gitlab.example.com/acme/site.git"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "backup.zip"), []byte("zip-content"), 0o600); err != nil {
		t.Fatal(err)
	}

	stub := &stubTransport{status: status}
	originalTransport := http.DefaultTransport
	http.DefaultTransport = stub
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
		rootCmd.SetArgs(nil)
	})

	// Flag values persist between Execute calls in one process.
	gitlabToken, gitlabPackageName, gitlabPackageVersion = "", "", ""

	rootCmd.SetArgs(append([]string{"gitlab:upload", "backup.zip"}, args...))
	return stub, rootCmd.Execute()
}

func TestGitlabUploadTokenPrecedence(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		gitlabEnv  string
		ciJobEnv   string
		wantHeader string
	}{
		{"--token wins over environment", []string{"--token", "from-flag"}, "from-gitlab", "from-ci", "from-flag"},
		{"-t wins over environment", []string{"-t", "from-short"}, "from-gitlab", "from-ci", "from-short"},
		{"GITLAB_TOKEN wins over CI_JOB_TOKEN", nil, "from-gitlab", "from-ci", "from-gitlab"},
		{"CI_JOB_TOKEN is the fallback", nil, "", "from-ci", "from-ci"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITLAB_TOKEN", tt.gitlabEnv)
			t.Setenv("CI_JOB_TOKEN", tt.ciJobEnv)

			stub, err := runUpload(t, http.StatusCreated, tt.args...)
			if err != nil {
				t.Fatalf("upload failed: %v", err)
			}
			if len(stub.requests) != 1 {
				t.Fatalf("got %d requests, want 1", len(stub.requests))
			}
			if got := stub.requests[0].Header.Get("PRIVATE-TOKEN"); got != tt.wantHeader {
				t.Errorf("PRIVATE-TOKEN = %q, want %q", got, tt.wantHeader)
			}
		})
	}
}

func TestGitlabUploadFailsWithoutTokenBeforeSending(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("CI_JOB_TOKEN", "")

	stub, err := runUpload(t, http.StatusCreated)
	if err == nil || !strings.Contains(err.Error(), "no GitLab token") {
		t.Fatalf("err = %v, want missing token error", err)
	}
	if len(stub.requests) != 0 {
		t.Errorf("got %d requests, want none", len(stub.requests))
	}
}

func TestGitlabUploadUsesPackageFlags(t *testing.T) {
	stub, err := runUpload(t, http.StatusCreated, "-t", "x", "--package-name", "my-app", "--package-version", "1.2.3")
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	req := stub.requests[0]
	want := "https://gitlab.example.com/api/v4/projects/acme%2Fsite/packages/generic/my-app/1.2.3/backup.zip"
	if got := req.URL.String(); got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
	if req.Method != http.MethodPut {
		t.Errorf("method = %q, want PUT", req.Method)
	}
	if stub.bodies[0] != "zip-content" {
		t.Errorf("body = %q, want the file content", stub.bodies[0])
	}
}

func TestGitlabUploadPackageDefaults(t *testing.T) {
	stub, err := runUpload(t, http.StatusCreated, "-t", "x")
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	want := regexp.MustCompile(`/packages/generic/site/\d{8}T\d{6}/backup\.zip$`)
	if got := stub.requests[0].URL.String(); !want.MatchString(got) {
		t.Errorf("URL = %q, want match for %s", got, want)
	}
}

func TestGitlabUploadReturnsServerError(t *testing.T) {
	_, err := runUpload(t, http.StatusForbidden, "-t", "x")
	if err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("err = %v, want HTTP 403 error", err)
	}
}
