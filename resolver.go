package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type semver struct {
	major int
	minor int
	patch int
}

func parseSemver(v string) (semver, bool) {
	re := regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(v)
	if matches == nil {
		return semver{}, false
	}
	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])
	return semver{major, minor, patch}, true
}

func resolveBestVersion(versions []string, constraint string) string {
	var parsed []semver
	for _, v := range versions {
		if sv, ok := parseSemver(v); ok {
			parsed = append(parsed, sv)
		}
	}

	sort.Slice(parsed, func(i, j int) bool {
		if parsed[i].major != parsed[j].major {
			return parsed[i].major < parsed[j].major
		}
		if parsed[i].minor != parsed[j].minor {
			return parsed[i].minor < parsed[j].minor
		}
		return parsed[i].patch < parsed[j].patch
	})

	constraint = strings.TrimSpace(constraint)

	var minVer, maxVer semver
	hasMin := false
	hasMax := false

	if strings.HasPrefix(constraint, "^") {
		base := strings.TrimPrefix(constraint, "^")
		if sv, ok := parseSemver(base); ok {
			minVer = sv
			maxVer = semver{sv.major + 1, 0, 0}
			hasMin = true
			hasMax = true
		}
	} else if strings.HasPrefix(constraint, "~") {
		base := strings.TrimPrefix(constraint, "~")
		if sv, ok := parseSemver(base); ok {
			minVer = sv
			maxVer = semver{sv.major, sv.minor + 1, 0}
			hasMin = true
			hasMax = true
		}
	} else if strings.HasPrefix(constraint, ">=") {
		base := strings.TrimPrefix(constraint, ">=")
		if sv, ok := parseSemver(base); ok {
			minVer = sv
			hasMin = true
		}
	} else if strings.HasPrefix(constraint, ">") {
		base := strings.TrimPrefix(constraint, ">")
		if sv, ok := parseSemver(base); ok {
			minVer = sv
			minVer.patch++
			hasMin = true
		}
	} else if strings.HasPrefix(constraint, "<=") {
		base := strings.TrimPrefix(constraint, "<=")
		if sv, ok := parseSemver(base); ok {
			maxVer = sv
			maxVer.patch++
			hasMax = true
		}
	} else if strings.HasPrefix(constraint, "<") {
		base := strings.TrimPrefix(constraint, "<")
		if sv, ok := parseSemver(base); ok {
			maxVer = sv
			hasMax = true
		}
	} else {
		if sv, ok := parseSemver(constraint); ok {
			minVer = sv
			maxVer = semver{sv.major, sv.minor, sv.patch + 1}
			hasMin = true
			hasMax = true
		}
	}

	if !hasMin && !hasMax {
		return ""
	}

	for i := len(parsed) - 1; i >= 0; i-- {
		v := parsed[i]
		if hasMin && (v.major < minVer.major || v.minor < minVer.minor || v.patch < minVer.patch) {
			continue
		}
		if hasMin && v.major == minVer.major && v.minor == minVer.minor && v.patch < minVer.patch {
			continue
		}
		if hasMin && v.major == minVer.major && v.minor < minVer.minor {
			continue
		}
		if hasMax && v.major >= maxVer.major {
			if v.major > maxVer.major {
				continue
			}
			if v.minor >= maxVer.minor {
				continue
			}
		}
		sv := v
		return fmt.Sprintf("%d.%d.%d", sv.major, sv.minor, sv.patch)
	}

	return ""
}

func findLatestVersion(versions []string) string {
	var parsed []semver
	for _, v := range versions {
		if sv, ok := parseSemver(v); ok {
			parsed = append(parsed, sv)
		}
	}
	if len(parsed) == 0 {
		return ""
	}
	sort.Slice(parsed, func(i, j int) bool {
		if parsed[i].major != parsed[j].major {
			return parsed[i].major < parsed[j].major
		}
		if parsed[i].minor != parsed[j].minor {
			return parsed[i].minor < parsed[j].minor
		}
		return parsed[i].patch < parsed[j].patch
	})
	best := parsed[len(parsed)-1]
	return fmt.Sprintf("%d.%d.%d", best.major, best.minor, best.patch)
}
