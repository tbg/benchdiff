package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// getRefAsSHA returns the provided git ref as a SHA.
func getRefAsSHA(ref string) (string, error) {
	ref, err := capture("git", "rev-parse", ref)
	if err != nil {
		return "", errors.Wrap(err, "getting git ref as sha")
	}
	return ref, nil
}

// getCurRef returns the active git ref in the current working directory's
// repository.
func getCurRef() (string, error) {
	ref, err := getRefAsSHA("HEAD")
	if err != nil {
		return "", errors.Wrap(err, "getting current git ref")
	}
	return ref, nil
}

// getCurRef returns the previous git ref in the current working directory's
// repository.
func getPrevRef(ref string) (string, error) {
	ref, err := getRefAsSHA(ref + "~")
	if err != nil {
		return "", errors.Wrap(err, "getting previous git ref")
	}
	return ref, nil
}

// getCurSymbolicRef returns the active git symbolic ref in the current working
// directory's repository. If a symbolic reference could not be found, returns
// false instead.
func getCurSymbolicRef() (string, bool, error) {
	ref, err := capture("git", "symbolic-ref", "HEAD")
	if err != nil {
		if strings.Contains(err.Error(), "not a symbolic ref") {
			return "", false, nil
		}
		return "", false, errors.Wrap(err, "getting current git ref")
	}
	ref = strings.TrimPrefix(ref, "refs/heads/")
	return ref, true, nil
}

// checkValidRef determines whether the provided git ref is valid in the current
// working directory's repository.
func checkValidRef(ref string) (bool, error) {
	_, err := capture("git", "cat-file", "-t", ref)
	if err != nil {
		if strings.Contains(err.Error(), "Not a valid object name") {
			return false, nil
		}
		return false, errors.Wrap(err, "checking valid ref")
	}
	return true, nil
}

// shortenRef attempts to shorten the git ref.
func shortenRef(ref string) string {
	if len(ref) <= 7 {
		return ref
	}
	shortRef := ref[:7]
	if _, err := strconv.ParseUint(shortRef, 16, 64); err != nil {
		// Not a SHA.
		return ref
	}
	if ok, err := checkValidRef(shortRef); ok && err == nil {
		return shortRef
	}
	return ref
}

// checkoutRef switches branches to the specified ref. If a post-checkout
// command is provided, it is run after checking out the ref.
func checkoutRef(ref string, postCheckout string) error {
	if err := spawn("git", "checkout", "-q", ref); err != nil {
		return errors.Wrap(err, "checkout ref")
	}
	if postCheckout == "" {
		return nil
	}
	args := strings.Split(postCheckout, " ")
	// Send all output of post-checkout hook to stderr.
	err := spawnWith(os.Stdin, os.Stderr, os.Stderr, args...)
	return errors.Wrap(err, "post-checkout")
}

func subjectForRef(ref string) (string, error) {
	return capture("git", "log", "--format=%s", "-1", ref)
}
