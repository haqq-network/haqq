{}:
let
  mkf = builtins.readFile ../Makefile;
  match = builtins.match ".+[\n]VERSION := \"([^\"]+)\".+" mkf;
in
builtins.head match
