{
  description = "Arboreal — Go development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go              # Go toolchain
            gopls           # Go LSP server
            gotools         # goimports, etc.
            go-tools        # staticcheck
            golangci-lint   # meta linter
            delve           # debugger (dlv)
            gomodifytags    # struct tag editing
            gotests         # test generation
          ];

          shellHook = ''
            echo "Go $(go version | cut -d' ' -f3) — gopls $(gopls version | head -n1)"
          '';
        };
      });
}
