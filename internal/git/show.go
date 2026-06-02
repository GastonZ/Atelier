package git

// Show returns the output of `git show --stat <hash>` verbatim.
// Returns an error if git exits non-zero.
// Note: Show is a method on execLogReader (same LogReader interface).
func (r *execLogReader) Show(repoPath, hash string) (string, error) {
	cmd := execCommand("git", "show", "--stat", hash)
	cmd.Dir = repoPath

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
