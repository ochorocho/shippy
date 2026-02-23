package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"shippy/internal/ui"
	"shippy/internal/upload"
)

var (
	gitlabToken          string
	gitlabPackageName    string
	gitlabPackageVersion string
)

var gitlabUploadCmd = &cobra.Command{
	Use:   "gitlab:upload <file>",
	Short: "Upload a file to GitLab Generic Packages",
	Long: `Upload a backup ZIP (or any file) to the GitLab Generic Packages
registry of the project's origin repository.

The GitLab host and project path are automatically detected from the
git remote "origin" URL. Authentication requires a token provided via
the --token flag, GITLAB_TOKEN, or CI_JOB_TOKEN environment variable.

Example:
  shippy gitlab:upload backup-production-20240109T120000.zip
  shippy gitlab:upload backup.zip --token glpat-xxxx
  shippy gitlab:upload backup.zip --package-name my-app --package-version 1.0.0`,
	Args:         cobra.ExactArgs(1),
	RunE:         runGitlabUpload,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(gitlabUploadCmd)
	gitlabUploadCmd.Flags().StringVarP(&gitlabToken, "token", "t", "", "GitLab token (default: $GITLAB_TOKEN or $CI_JOB_TOKEN)")
	gitlabUploadCmd.Flags().StringVar(&gitlabPackageName, "package-name", "", "Package name (default: project name from git remote)")
	gitlabUploadCmd.Flags().StringVar(&gitlabPackageVersion, "package-version", "", "Package version (default: timestamp)")
}

func runGitlabUpload(cmd *cobra.Command, args []string) error {
	out := ui.New()
	filePath := args[0]

	out.Header("Shippy - GitLab Upload")

	// Verify file exists
	if _, err := os.Stat(filePath); err != nil {
		out.Error("File not found: %s", filePath)
		return fmt.Errorf("file not found: %s", filePath)
	}

	// Step 1: Detect git remote
	out.StepNumber(1, "Detecting git remote")

	remoteInfo, err := upload.ParseGitRemoteFromRepo(".")
	if err != nil {
		out.Error("Failed to detect git remote: %v", err)
		return err
	}

	out.Success("Project: %s (on %s)", remoteInfo.ProjectPath, remoteInfo.Host)

	// Step 2: Resolve authentication
	out.StepNumber(2, "Resolving authentication")

	token := gitlabToken
	if token == "" {
		token = os.Getenv("GITLAB_TOKEN")
	}
	if token == "" {
		token = os.Getenv("CI_JOB_TOKEN")
	}
	if token == "" {
		out.Error("No GitLab token found. Set --token, GITLAB_TOKEN, or CI_JOB_TOKEN")
		return fmt.Errorf("no GitLab token provided")
	}

	out.Success("Token configured")

	// Step 3: Upload
	out.StepNumber(3, "Uploading to GitLab Packages")

	packageName := gitlabPackageName
	if packageName == "" {
		packageName = remoteInfo.ProjectName()
	}

	packageVersion := gitlabPackageVersion
	if packageVersion == "" {
		packageVersion = time.Now().Format("20060102T150405")
	}

	uploader := upload.NewGitLabUploader(remoteInfo.Host, remoteInfo.ProjectPath, token)
	result, err := uploader.Upload(filePath, upload.UploadOptions{
		PackageName:    packageName,
		PackageVersion: packageVersion,
	})
	if err != nil {
		out.Error("Upload failed: %v", err)
		return err
	}

	out.Success(result.Message)
	out.Info("Packages: %s", result.URL)

	out.HeaderGreen("Upload completed successfully!")

	return nil
}
