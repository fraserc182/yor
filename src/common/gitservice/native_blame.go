package gitservice

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// nativeGitBlame shells out to native git to compute blame for a file.
// This is 10-100x faster than go-git's in-process Blame() implementation
// and is safe to call concurrently (each invocation is a separate process).
func nativeGitBlame(repoRootDir string, relativeFilePath string, revision string) (*git.BlameResult, error) {
	cmd := exec.Command("git", "blame", "--porcelain", revision, "--", relativeFilePath)
	cmd.Dir = repoRootDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git blame failed for %s at %s: %w", relativeFilePath, revision, err)
	}

	return parsePorcelainBlame(output, relativeFilePath)
}

// parsePorcelainBlame parses the output of `git blame --porcelain` into a git.BlameResult.
//
// Porcelain format per line:
//
//	<hash> <orig-line> <final-line> [<num-lines>]
//	[header fields for first occurrence of this commit...]
//	\t<content line>
//
// Header fields include: author, author-mail, author-time, author-tz, etc.
// Only the first occurrence of a commit in the output includes the full headers.
func parsePorcelainBlame(output []byte, filePath string) (*git.BlameResult, error) {
	result := &git.BlameResult{
		Path:  filePath,
		Lines: make([]*git.Line, 0),
	}

	// Cache commit metadata so we don't re-parse for repeated commits
	type commitInfo struct {
		hash       plumbing.Hash
		authorName string
		authorMail string
		authorTime time.Time
	}
	commits := make(map[plumbing.Hash]*commitInfo)

	scanner := bufio.NewScanner(bytes.NewReader(output))

	var currentCommit *commitInfo
	var currentHash plumbing.Hash

	for scanner.Scan() {
		line := scanner.Text()

		// Content lines start with a tab
		if strings.HasPrefix(line, "\t") {
			if currentCommit == nil {
				return nil, fmt.Errorf("unexpected content line without commit header")
			}
			result.Lines = append(result.Lines, &git.Line{
				Author:     currentCommit.authorMail,
				AuthorName: currentCommit.authorName,
				Text:       line[1:], // strip the leading tab
				Date:       currentCommit.authorTime,
				Hash:       currentCommit.hash,
			})
			currentCommit = nil
			continue
		}

		// Check if this is a commit line: 40-hex-char hash followed by line numbers
		if len(line) >= 40 && isHexString(line[:40]) {
			hashStr := line[:40]
			currentHash = plumbing.NewHash(hashStr)

			if ci, ok := commits[currentHash]; ok {
				// We've seen this commit before, reuse its metadata
				currentCommit = ci
			} else {
				// New commit, will be populated by subsequent header lines
				currentCommit = &commitInfo{hash: currentHash}
				commits[currentHash] = currentCommit
			}
			continue
		}

		// Header lines for the current commit
		if currentCommit != nil {
			switch {
			case strings.HasPrefix(line, "author-mail "):
				// Format: author-mail <email@example.com>
				mail := strings.TrimPrefix(line, "author-mail ")
				mail = strings.TrimPrefix(mail, "<")
				mail = strings.TrimSuffix(mail, ">")
				currentCommit.authorMail = mail
			case strings.HasPrefix(line, "author-time "):
				timeStr := strings.TrimPrefix(line, "author-time ")
				unix, err := strconv.ParseInt(timeStr, 10, 64)
				if err == nil {
					currentCommit.authorTime = time.Unix(unix, 0).UTC()
				}
			case strings.HasPrefix(line, "author "):
				currentCommit.authorName = strings.TrimPrefix(line, "author ")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning git blame output: %w", err)
	}

	return result, nil
}

// isHexString checks if a string contains only hexadecimal characters.
func isHexString(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return len(s) > 0
}
