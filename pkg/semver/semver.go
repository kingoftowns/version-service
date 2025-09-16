package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var semverRegex = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-(.+))?$`)

type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

func Parse(version string) (*Version, error) {
	matches := semverRegex.FindStringSubmatch(version)
	if matches == nil {
		return nil, fmt.Errorf("invalid semantic version: %s", version)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])
	prerelease := matches[4]

	return &Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
	}, nil
}

func (v *Version) String() string {
	base := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		return fmt.Sprintf("%s-%s", base, v.Prerelease)
	}
	return base
}

func (v *Version) IncrementPatch() *Version {
	return &Version{
		Major: v.Major,
		Minor: v.Minor,
		Patch: v.Patch + 1,
	}
}

func (v *Version) IncrementMinor() *Version {
	return &Version{
		Major: v.Major,
		Minor: v.Minor + 1,
		Patch: 0,
	}
}

func (v *Version) IncrementMajor() *Version {
	return &Version{
		Major: v.Major + 1,
		Minor: 0,
		Patch: 0,
	}
}

func (v *Version) WithDevSuffix(sha string) *Version {
	shortSHA := sha
	if len(sha) > 7 {
		shortSHA = sha[:7]
	}
	return &Version{
		Major:      v.Major,
		Minor:      v.Minor,
		Patch:      v.Patch,
		Prerelease: fmt.Sprintf("dev-%s", shortSHA),
	}
}

func IsValid(version string) bool {
	_, err := Parse(version)
	return err == nil
}

func Compare(v1, v2 string) (int, error) {
	version1, err := Parse(v1)
	if err != nil {
		return 0, err
	}
	version2, err := Parse(v2)
	if err != nil {
		return 0, err
	}

	if version1.Major != version2.Major {
		return version1.Major - version2.Major, nil
	}
	if version1.Minor != version2.Minor {
		return version1.Minor - version2.Minor, nil
	}
	if version1.Patch != version2.Patch {
		return version1.Patch - version2.Patch, nil
	}

	if version1.Prerelease == "" && version2.Prerelease != "" {
		return 1, nil
	}
	if version1.Prerelease != "" && version2.Prerelease == "" {
		return -1, nil
	}

	return strings.Compare(version1.Prerelease, version2.Prerelease), nil
}