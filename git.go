package git

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
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

type Oid string

// Returns stdout, stderr, error
func runGit(repoPath string, printError bool, arg ...string) (string, string, error) {
	path, err := getGitPath()
	if err != nil {
		return "", "", err
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
		switch err.(type) {
		case *exec.ExitError:

		default:
			printError = true
		}
		if printError {
			fmt.Fprintf(os.Stderr, "%s", errstr)
		}
		return outstr, errstr, err
	}

	return outstr, errstr, nil
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
		return StatusFlagUnmergedUpdated, nil
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
	out, _, err := runGit(path, true, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}

	return &Repo{Path: strings.TrimSuffix(out, "\n")}, nil
}

func (r *Repo) RevParse(obj string) (Oid, error) {
	out, _, err := runGit(r.Path, true, "rev-parse", obj)
	if err != nil {
		return "", err
	}
	return Oid(strings.TrimSuffix(out, "\n")), nil
}

func (r *Repo) RevParseAbbrev(obj string) (string, error) {
	out, _, err := runGit(r.Path, true, "rev-parse", "--abbrev-ref", obj)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(out, "\n"), nil
}

func (r *Repo) Status() ([]status, error) {
	out, _, err := runGit(r.Path, true, "status", "--porcelain", "-uno")
	if err != nil {
		return nil, err
	}

	ss, err := parseStatus(out)
	if err != nil {
		return nil, err
	}

	return ss, nil
}

type State int

const (
	StateNone State = iota
	StateRebaseInteractive
	StateRebaseMerge
	StateRebase
	StateApplyMailbox
	StateApplyMailboxOrRebase
	StateMerge
	StateRevert
	StateCherryPick
	StateBisect
)

func (r *Repo) gitFilePath(name string) string {
	return path.Join(r.Path, ".git", name)
}

func (r *Repo) HasGitFile(name string) (bool, error) {
	_, err := os.Stat(r.gitFilePath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repo) RemoveGitFile(name string) error {
	return os.Remove(r.gitFilePath(name))
}

var stateFiles map[string]State = map[string]State{
	path.Join("rebase-merge", "interactive"): StateRebaseInteractive,
	"rebase-merge":                           StateRebaseMerge,
	path.Join("rebase-apply", "rebasing"):    StateRebase,
	path.Join("rebase-apply", "applying"):    StateApplyMailbox,
	"rebase-apply":                           StateApplyMailboxOrRebase,
	"MERGE_HEAD":                             StateMerge,
	"REVERT_HEAD":                            StateRevert,
	"CHERRY_PICK_HEAD":                       StateCherryPick,
	"BISECT_LOG":                             StateBisect,
}

func (r *Repo) State() (State, error) {
	for file, state := range stateFiles {
		b, err := r.HasGitFile(file)
		if err != nil {
			return 0, err
		}
		if b {
			return state, nil
		}
	}
	return StateNone, nil
}

func (r *Repo) Add(path string) error {
	_, _, err := runGit(r.Path, true, "add", "--", path)
	return err
}

func (r *Repo) CommitReuse(original Oid) error {
	_, _, err := runGit(r.Path, true, "commit", "-C", string(original), "--no-edit", "--allow-empty")
	return err
}

func (r *Repo) CommitAmend() error {
	_, _, err := runGit(r.Path, true, "commit", "--amend", "--no-edit", "--allow-empty")
	return err
}

func (r *Repo) ResetHard(commit Oid) error {
	_, _, err := runGit(r.Path, true, "reset", "--hard", string(commit))
	return err
}

func (r *Repo) CherryPickHead() (Oid, error) {
	bytes, err := ioutil.ReadFile(r.gitFilePath("CHERRY_PICK_HEAD"))
	if err != nil {
		return "", err
	}
	return Oid(strings.TrimSpace(string(bytes))), nil
}

// Returns whether the cherry-pick succeeded without conflicts.
func (r *Repo) CherryPick(commit Oid) (bool, error) {
	_, stderr, err := runGit(r.Path, false, "cherry-pick", "--allow-empty", string(commit))
	if err != nil {
		state, stateErr := r.State()
		if stateErr != nil {
			fmt.Fprintf(os.Stderr, "%s", stderr)
			return false, err
		}
		if state == StateCherryPick {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repo) CherryPickContinue() error {
	_, _, err := runGit(r.Path, true, "cherry-pick", "--continue")
	return err
}

func (r *Repo) Parents(commit Oid) ([]Oid, error) {
	out, _, err := runGit(r.Path, true, "show", "--raw", "--no-patch", "--format=format:%P", string(commit))
	if err != nil {
		return nil, err
	}

	commits := []Oid{}
	for _, c := range strings.Fields(out) {
		commits = append(commits, Oid(c))
	}

	return commits, nil
}
