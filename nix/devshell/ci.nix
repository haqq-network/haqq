{ ... }:
{
  git-hooks.enable = false;

  scripts.ci-check-version.exec = ''
    set -e
    export NIX_CONFIG="experimental-features = nix-command flakes"
    MAKEFILE_VERSION=$(grep "^VERSION :=" Makefile | awk -F '"' '{print $2}')
    FLAKE_VERSION=$(nix derivation show .#haqq 2>/dev/null | jq -r '.[].env.version' || echo "")

    if [[ -z "$FLAKE_VERSION" ]]; then
      echo "Error: Could not determine flake version. Make sure Nix flakes are enabled."
      exit 1
    fi

    if [[ "$MAKEFILE_VERSION" != "$FLAKE_VERSION" ]]; then
      echo "Error: Makefile version ($MAKEFILE_VERSION) and haqqd package version ($FLAKE_VERSION) are not equal."
      echo "Please update version in ./nix/package.nix or Makefile to match."
      exit 1
    fi
    
    echo "Version check passed: $MAKEFILE_VERSION"
  '';

  scripts.ci-check-gomod2nix.exec = ''
    set -e
    export NIX_CONFIG="experimental-features = nix-command flakes"
    
    if ! command -v gomod2nix &> /dev/null; then
      echo "Error: gomod2nix is not available in the environment"
      exit 1
    fi
    
    echo "Generating gomod2nix.toml..."
    gomod2nix generate
    
    if ! git diff --exit-code; then
      echo "Error: Directory is not clean after gomod2nix generation"
      echo "Please run 'gomod2nix generate' and commit the changes to gomod2nix.toml"
      git diff --stat
      exit 1
    fi
    
    echo "gomod2nix check passed: gomod2nix.toml is up to date"
  '';

  scripts.ci-proto.exec = ''
    set -e
    export NIX_CONFIG="experimental-features = nix-command flakes"

    echo "Cleaning previous builds..."
    make clean || true

    echo "Generating protobuf files..."
    make proto-all

    echo "Generating swagger documentation..."
    make proto-swagger-gen

    # These files get updated every time due to non-deterministic generation, so we ignore them
    git checkout -- client/docs/statik/statik.go client/docs/swagger-ui/swagger.json || true

    echo "Checking for uncommitted changes in generated files..."
    
    # Check for changes in proto-generated files only
    # Exclude swagger.json and statik.go as they are non-deterministic and get regenerated each time
    CHANGED_FILES=$(git diff --name-only | grep -E '\.(pb\.go|pb\.gw\.go|pulsar\.go)$' || true)
    
    if [ -n "$CHANGED_FILES" ]; then
      echo "Error: Directory is not clean after proto/swagger generation"
      echo "The following generated files have been modified:"
      echo "$CHANGED_FILES"
      echo ""
      echo "Please run 'make proto-all && make proto-swagger-gen' and commit the changes"
      exit 1
    fi
    
    echo "Proto check passed: all generated files are committed"
  '';
}
