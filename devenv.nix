{ pkgs, ... }:

{
  # https://devenv.sh/basics/
  env.GREET = "devenv";

  # https://devenv.sh/packages/
  packages = with pkgs; [ 
    gcc
    go_1_19
    git
    ];

  enterShell = ''
    git --version
    go version
  '';

  # https://devenv.sh/languages/
  languages.nix.enable = true;

  # Using the languages atributeset we are unable to set a version
  #languages.go.enable = true;

  # https://devenv.sh/scripts/
  # scripts.hello.exec = "echo hello from $GREET";

  # https://devenv.sh/pre-commit-hooks/
  # pre-commit.hooks.shellcheck.enable = true;

  # https://devenv.sh/processes/
  # processes.ping.exec = "ping example.com";

}
