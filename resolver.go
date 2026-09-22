package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type semver struct { major, minor, patch int }

var semverRE = regexp.MustCompile(`^v?(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

func parseSemver(v string) (semver, bool) {
	m := semverRE.FindStringSubmatch(strings.TrimSpace(v))
	if m == nil { return semver{}, false }
	major, _ := strconv.Atoi(m[1]); minor, patch := 0, 0
	if m[2] != "" { minor, _ = strconv.Atoi(m[2]) }
	if m[3] != "" { patch, _ = strconv.Atoi(m[3]) }
	return semver{major, minor, patch}, true
}

func cmpSemver(a, b semver) int {
	if a.major != b.major { if a.major < b.major { return -1 }; return 1 }
	if a.minor != b.minor { if a.minor < b.minor { return -1 }; return 1 }
	if a.patch < b.patch { return -1 }; if a.patch > b.patch { return 1 }; return 0
}

func wildcardConstraint(raw string) (semver, int, bool) {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "v"))
	parts := strings.Split(raw, ".")
	if len(parts) > 3 { return semver{}, 0, false }
	wild := -1
	values := [3]int{}
	for i := range parts {
		p := strings.ToLower(parts[i])
		if p == "x" || p == "*" { wild = i; break }
		n, err := strconv.Atoi(p); if err != nil { return semver{}, 0, false }; values[i] = n
	}
	if wild < 0 && len(parts) < 3 { wild = len(parts) }
	if wild < 0 { return semver{values[0], values[1], values[2]}, 3, true }
	return semver{values[0], values[1], values[2]}, wild, true
}

func satisfiesComparator(v semver, token string) bool {
	token = strings.TrimSpace(token)
	if token == "" || token == "*" || strings.EqualFold(token, "latest") { return true }
	if strings.HasPrefix(token, "^") {
		base, ok := parseSemver(strings.TrimSpace(token[1:])); if !ok { return false }
		upper := semver{base.major + 1, 0, 0}; if base.major == 0 { upper = semver{0, base.minor + 1, 0}; if base.minor == 0 { upper = semver{0, 0, base.patch + 1} } }
		return cmpSemver(v, base) >= 0 && cmpSemver(v, upper) < 0
	}
	if strings.HasPrefix(token, "~") {
		base, ok := parseSemver(strings.TrimSpace(strings.TrimLeft(token[1:], " "))); if !ok { return false }
		return cmpSemver(v, base) >= 0 && cmpSemver(v, semver{base.major, base.minor + 1, 0}) < 0
	}
	for _, op := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(token, op) {
			base, ok := parseSemver(strings.TrimSpace(token[len(op):])); if !ok { return false }
			c := cmpSemver(v, base); switch op { case ">=": return c >= 0; case "<=": return c <= 0; case ">": return c > 0; case "<": return c < 0; default: return c == 0 }
		}
	}
	base, precision, ok := wildcardConstraint(token); if !ok { return false }
	if precision == 3 { return cmpSemver(v, base) == 0 }
	if precision == 0 { return v.major == 0 }
	if precision == 1 { return v.major == base.major }
	return v.major == base.major && v.minor == base.minor
}

func satisfiesRange(v semver, constraint string) bool {
	for _, alternative := range strings.Split(constraint, "||") {
		alternative = strings.TrimSpace(alternative)
		if strings.Contains(alternative, " - ") {
			p := strings.SplitN(alternative, " - ", 2); a, aok := parseSemver(strings.TrimSpace(p[0])); b, bok := parseSemver(strings.TrimSpace(p[1])); if aok && bok && cmpSemver(v,a) >= 0 && cmpSemver(v,b) <= 0 { return true }; continue
		}
		ok := true
		for _, token := range strings.Fields(alternative) { if !satisfiesComparator(v, token) { ok = false; break } }
		if ok { return true }
	}
	return false
}

func resolveBestVersion(versions []string, constraint string) string {
	type candidate struct { raw string; parsed semver }
	candidates := make([]candidate, 0, len(versions))
	for _, raw := range versions { if v, ok := parseSemver(raw); ok && !strings.Contains(raw, "-") { candidates = append(candidates, candidate{raw, v}) } }
	sort.Slice(candidates, func(i,j int) bool { return cmpSemver(candidates[i].parsed, candidates[j].parsed) < 0 })
	for i := len(candidates)-1; i >= 0; i-- { if satisfiesRange(candidates[i].parsed, strings.TrimSpace(constraint)) { return candidates[i].raw } }
	return ""
}

func findLatestVersion(versions []string) string { return resolveBestVersion(versions, "*") }

var _ = fmt.Sprintf
