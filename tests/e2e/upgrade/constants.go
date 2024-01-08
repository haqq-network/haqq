package upgrade

// The constants used in the upgrade tests are defined here
const (
	// the defaultChainID used for testing
	defaultChainID = "haqq_11235-1"

	// LocalVersionTag defines the docker image ImageTag when building locally
	LocalVersionTag = "latest"

	// haqqRepo is the docker hub repository that contains the Haqq images pulled during tests
	haqqRepo = "alhaqq/haqq"

	// upgradesPath is the relative path from this folder to the app/upgrades folder
	upgradesPath = "../../../app/upgrades"

	// versionSeparator is used to separate versions in the INITIAL_VERSION and TARGET_VERSION
	// environment vars
	versionSeparator = "/"
)
