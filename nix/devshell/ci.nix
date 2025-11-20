{ ... }:
{
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

    # it gets updated every time, so we are ignoring this one
    git checkout -- client/docs/statik/statik.go || true

    echo "Checking for uncommitted changes..."
    
    if ! git diff --exit-code; then
      echo "Error: Directory is not clean after proto/swagger generation"
      echo "The following files have been modified:"
      git diff --name-only
      echo ""
      echo "Please run 'make proto-all && make proto-swagger-gen' and commit the changes"
      exit 1
    fi
    
    echo "Proto check passed: all generated files are committed"
  '';
}
