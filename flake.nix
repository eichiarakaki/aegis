{
  description = "Aegis Development Environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, utils }:
    utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
      in
      {
        devShells.default = pkgs.mkShell {
          # Core Development Tools
          buildInputs = with pkgs; [
            go_1_26
            golangci-lint # Industry-standard linter
            delve         # Essential for professional debugging
            gotools       # Includes godoc, goimports, etc.
            gopls         # Go Language Server (for VS Code/Neovim)

            nats-server
          ];

          # Productivity & Automation (DX)
          nativeBuildInputs = with pkgs; [
            air           # Hot reload for Go apps
            gh            # GitHub CLI
          ];

          # Environment Setup
          shellHook = ''
            export GOPATH="$HOME/go"
            export PATH="$GOPATH/bin:$PATH"
            
            clear
            
            echo "---------- Go 1.26 Environment -----------"
            go version
            echo "------------------------------------------"
          '';
        };
      });
}