package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var gitPath string

func getGitPath() (string, error) {
	if gitPath == "" {
		path, err := exec.LookPath("git")
		if err != nil {
			return "", err
		}
		gitPath = path
	}
	return gitPath, nil
}

type Repo struct {
	Path string
}

// Returns stdout, error
func runGit(repoPath string, arg ...string) (string, error) {
	path, err := getGitPath()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(path, arg...)
	cmd.Dir = repoPath

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	outstr := string(stdout.Bytes())
	errstr := string(stderr.Bytes())

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", errstr)
		return outstr, err
	}

	return outstr, nil
}

type StatusFlag int

const (
	StatusFlagUnmodified StatusFlag = iota
	StatusFlagModified
	StatusFlagAdded
	StatusFlagDeleted
	StatusFlagRenamed
	StatusFlagCopied
	StatusFlagUnmergedUpdated
)

type status struct {
	OldPath        string
	NewPath        string
	IndexStatus    StatusFlag
	WorkTreeStatus StatusFlag
}

func statusFlagForChar(c byte) (StatusFlag, error) {
	switch c {
	case ' ':
		return StatusFlagUnmodified, nil
	case 'M':
		return StatusFlagModified, nil
	case 'A':
		return StatusFlagAdded, nil
	case 'D':
		return StatusFlagDeleted, nil
	case 'R':
		return StatusFlagRenamed, nil
	case 'C':
		return StatusFlagCopied, nil
	case 'U':
		return StatusFlagUnmodified, nil
	default:
		return 0, fmt.Errorf("Unknown status flag `%c`", c)
	}
}

func parseStatus(output string) ([]status, error) {
	ss := []status{}
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		if len(line) < 4 {
			return nil, errors.New("Status line too short")
		}

		indexStatus, err := statusFlagForChar(line[0])
		if err != nil {
			return nil, err
		}
		workTreeStatus, err := statusFlagForChar(line[1])
		if err != nil {
			return nil, err
		}

		paths := strings.SplitN(line[3:], " -> ", 2)
		var newPath string
		if len(paths) == 2 {
			newPath = paths[1]
		}

		s := status{
			OldPath:        paths[0],
			NewPath:        newPath,
			IndexStatus:    indexStatus,
			WorkTreeStatus: workTreeStatus,
		}
		ss = append(ss, s)
	}

	return ss, nil
}

func Repository(path string) (*Repo, error) {
	out, err := runGit(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}

	return &Repo{Path: strings.TrimSuffix(out, "\n")}, nil
}

func (r *Repo) Status() ([]status, error) {
	out, err := runGit(r.Path, "status", "--porcelain", "-uno")
	if err != nil {
		return nil, err
	}

	ss, err := parseStatus(out)
	if err != nil {
		return nil, err
	}

	return ss, nil
}
