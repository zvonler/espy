package utils

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func TrimmedURL(url *url.URL) *url.URL {
	if strings.HasSuffix(url.RequestURI(), "/") {
		// Eliminate trailing slashes to canonicalize URL for database
		if trimmed, err := url.Parse(strings.TrimRight(url.String(), "/")); err != nil {
			panic(fmt.Sprintf("Bad URL: %v", err))
		} else {
			return trimmed
		}
	}
	return url
}

func PathExists(path string) (res bool, err error) {
	_, statErr := os.Stat(path)
	if statErr == nil {
		res = true
	} else if !os.IsNotExist(statErr) {
		err = statErr
	}
	return
}

func ParseURLOrID(arg string) (url *url.URL, id uint, err error) {
	var digitCheck = regexp.MustCompile(`^[0-9]+$`)
	if digitCheck.MatchString(arg) {
		if val, err := strconv.Atoi(arg); err == nil {
			id = uint(val)
		}
	} else {
		url, err = url.Parse(arg)
	}
	return
}
