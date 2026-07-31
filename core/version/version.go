package version

var (
    // Version platform versiyonu
    Version = "0.5.0"
    
    // BuildTime build zamanı
    BuildTime = ""
    
    // GitCommit git commit hash
    GitCommit = ""
    
    // GitBranch git branch
    GitBranch = ""
)

// GetVersion versiyon bilgisini döndürür
func GetVersion() string {
    if GitCommit != "" {
        return Version + " (" + GitCommit[:8] + ")"
    }
    return Version
}

// GetFullVersion tam versiyon bilgisini döndürür
func GetFullVersion() map[string]string {
    return map[string]string{
        "version":    Version,
        "build_time": BuildTime,
        "git_commit": GitCommit,
        "git_branch": GitBranch,
    }
}
