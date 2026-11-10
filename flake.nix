{
  description = "Timed coding practice in the terminal";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = { nixpkgs, ... }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
    in {
      packages = nixpkgs.lib.genAttrs systems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in {
          default = (pkgs.buildGoModule.override { go = pkgs.go_1_26; }) {
            pname = "codedrill";
            version = "0-unstable";
            src = ./.;
            vendorHash = "sha256-XlWdVg367vx3edpQ3y1tK+EZ34dKgjbWlf9TGdR0L3o=";
            subPackages = [ "cmd/codedrill" ];
            postPatch = ''
              substituteInPlace internal/runner/{runner.go,match_test.go,reference_test.go} \
                cmd/codedrill/{reference_pane_test.go,reference_view_test.go} \
                --replace-fail '#!/usr/bin/env bash' '#!${pkgs.runtimeShell}'
            '';
            meta = {
              description = "Timed coding practice in the terminal";
              homepage = "https://github.com/luvcie/codedrill";
              license = pkgs.lib.licenses.mit;
              mainProgram = "codedrill";
            };
          };
        });
    };
}
