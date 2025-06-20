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

        go = pkgs.go_1_24;

        # Plugins and other tools
        tools = with pkgs; [
          go
          protobuf
          protoc-gen-go
          protoc-gen-go-grpc
          ko
          kind
          bun
          nodejs_22          
        ];

      in {
        devShells.default = pkgs.mkShell {
          buildInputs = tools;

          # Set English locale for all tools
          LANG = "en_US.UTF-8";
          LC_ALL = "en_US.UTF-8";

          shellHook = ''            
            set -a
            if [ -f .env.local ]; then
              echo "📄 Loading environment from .env.local"
              . .env.local
            else
              echo "⚠️  .env.local not found"
            fi
            set +a

            ./scripts/docker-login.sh
            ./scripts/setup-tools.sh

            echo "🚀 Development environment ready!"
          '';
        };
      });
}
