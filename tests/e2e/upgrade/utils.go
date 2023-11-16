package upgrade

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
)

// HaqqVersions is a custom comparator for sorting semver version strings.
type HaqqVersions []string

// Len is the number of stored versions..
func (v HaqqVersions) Len() int { return len(v) }

// Swap swaps the elements with indexes i and j. It is needed to sort the slice.
func (v HaqqVersions) Swap(i, j int) { v[i], v[j] = v[j], v[i] }

// Less compares semver versions strings properly
func (v HaqqVersions) Less(i, j int) bool {
	v1, err := version.NewVersion(v[i])
	if err != nil {
		log.Fatalf("couldn't interpret version as SemVer string: %s: %s", v[i], err.Error())
	}
	v2, err := version.NewVersion(v[j])
	if err != nil {
		log.Fatalf("couldn't interpret version as SemVer string: %s: %s", v[j], err.Error())
	}
	return v1.LessThan(v2)
}

// CheckLegacyProposal checks if the running node requires a legacy proposal
func CheckLegacyProposal(version string) bool {
	version = strings.TrimSpace(version)
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	// check if the version is lower than v10.x.x
	cmp := HaqqVersions([]string{version, "v10.0.0"})
	isLegacyProposal := !cmp.Less(0, 1)

	return isLegacyProposal
}

// RetrieveUpgradesList parses the app/upgrades folder and returns a slice of semver upgrade versions
// in ascending order, e.g ["v1.0.0", "v1.0.1", "v1.1.0", ... , "v10.0.0"]
func RetrieveUpgradesList(upgradesPath string) ([]string, error) {
	dirs, err := os.ReadDir(upgradesPath)
	if err != nil {
		return nil, err
	}

	// preallocate slice to store versions
	versions := []string{}

	// pattern to find quoted string(upgrade version) in a file e.g. "v10.0.0"
	pattern := regexp.MustCompile(`"(.*?)"`)

	for _, d := range dirs {
		if d.Name() == ".DS_Store" {
			// skip processing .DS_Store
			continue
		}

		// creating path to upgrade dir file with constant upgrade version
		constantsPath := fmt.Sprintf("%s/%s/constants.go", upgradesPath, d.Name())
		f, err := os.ReadFile(constantsPath)
		if err != nil {
			return nil, err
		}
		v := pattern.FindString(string(f))
		// v[1 : len(v)-1] subslice used to remove quotes from version string
		versions = append(versions, v[1:len(v)-1])
	}

	sort.Sort(HaqqVersions(versions))

	return versions, nil
}

// ExportState executes the  'docker cp' command to copy container .haqqd dir
// to the specified target dir (local)
//
// See https://docs.docker.com/engine/reference/commandline/cp/
func (m *Manager) ExportState(targetDir string) error {
	/* #nosec G204 */
	cmd := exec.Command(
		"docker",
		"cp",
		fmt.Sprintf("%s:/root/.haqqd", m.ContainerID()),
		targetDir,
	)
	return cmd.Run()
}
