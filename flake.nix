{
  description = "Dev environment for Futura";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };

        go = pkgs.go_1_22;

        # Plugins and other tools
        tools = with pkgs; [
          go
          protobuf
          protoc-gen-go
          protoc-gen-go-grpc
          ko
          kind
          bun
        ];

      in {
        devShells.default = pkgs.mkShell {
          buildInputs = tools;
          shellHook = ''
            echo "🚀 Development environment ready!"
            echo "🔧 Available: Go, protoc, protoc-gen-go, ko, kind, bun"
          '';
        };
      });
}
