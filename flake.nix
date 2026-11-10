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
            nativeBuildInputs = [ pkgs.makeWrapper ];
            nativeCheckInputs = [ pkgs.bash pkgs.stdenv.cc pkgs.python3 ];
            checkPhase = ''
              runHook preCheck
              go test ./...
              runHook postCheck
            '';
            postPatch = ''
              substituteInPlace internal/runner/{runner.go,match_test.go,reference_test.go} \
                cmd/codedrill/{reference_pane_test.go,reference_view_test.go} \
                --replace-fail '#!/usr/bin/env bash' '#!${pkgs.runtimeShell}'
              substituteInPlace internal/ui/reference_diff_test.go \
                --replace-fail '#!/bin/sh' '#!${pkgs.runtimeShell}' \
                --replace-fail 'exec /bin/sleep' 'exec ${pkgs.coreutils}/bin/sleep'
            '';
            postInstall = ''
              wrapProgram $out/bin/codedrill \
                --suffix PATH : ${pkgs.lib.makeBinPath [ pkgs.bash pkgs.coreutils pkgs.gnugrep pkgs.stdenv.cc pkgs.python3 ]}
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
