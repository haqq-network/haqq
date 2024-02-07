{ ... }:
{
  scripts.ci-check-version.exec = ''
    set -e
    MAKEFILE_VERSION=$(grep "^VERSION :=" Makefile | awk -F '"' '{print $2}')
    FLAKE_VERSION=$(nix derivation show .#haqq | jq -r '.[].env.version')

    if [[ "$MAKEFILE_VERSION" != "$FLAKE_VERSION" ]]; then
      echo "Makefile version ($MAKEFILE_VERSION) and haqqd package version ($FLAKE_VERSION) are not equal. Please update version in ./nix/package.nix"
      exit 1
    fi
  '';

  scripts.ci-check-gomod2nix.exec = ''
    set -e
    gomod2nix generate
    if ! git diff --exit-code; then
    echo "Directory is not clean after gomod2nix generation"
    echo "Please run gomod2nix and commit the changes"
    exit 1
    fi
  '';

  scripts.ci-proto.exec = ''
    set -e

    make clean
    make proto-all
    make proto-swagger-gen

    # it gets updated every time, so we are ignoring this one
    git checkout -- client/docs/statik/statik.go

    echo "Checking diff..."
    
    if ! git diff --exit-code; then
    echo "Directory is not clean after swagger generation"
    exit 1
    fi
  '';
}
