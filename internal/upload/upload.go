package upload

// Uploader defines the interface for uploading backup files to a package registry.
// Implementations exist for GitLab, with room for GitHub, Gitea, etc.
type Uploader interface {
	Upload(filePath string, opts UploadOptions) (*UploadResult, error)
}

// UploadOptions configures the upload
type UploadOptions struct {
	PackageName    string
	PackageVersion string
}

// UploadResult contains the result of an upload
type UploadResult struct {
	URL     string
	Message string
}
