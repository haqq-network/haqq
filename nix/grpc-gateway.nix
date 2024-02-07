{ pkgs ? import <nixpkgs> { } }:
with pkgs;
buildGoModule rec {
  pname = "grpc-gateway";
  version = "2.18.1";

  src = fetchFromGitHub {
    owner = "grpc-ecosystem";
    repo = "grpc-gateway";
    rev = "v${version}";
    sha256 = "sha256-mbRceXqc7UmrhM2Y6JJIUvMf9YxMFMjRW7VvEa8/xHs=";
  };

  vendorHash = "sha256-zVojs4q8TytJY3myKvLdACnMFJ0iK9Cfn+aZ4d/j34s=";

  nativeBuildInputs = [ pkgs.installShellFiles ];

  subPackages = [ "protoc-gen-grpc-gateway" "protoc-gen-openapiv2" ];
}
