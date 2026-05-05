{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    (flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config = {
            allowUnfreePredicate = pkg: pkg.pname == "ngrok";
          };
        };

        readmeGeneratorForHelm = pkgs.buildNpmPackage {
          pname = "readme-generator-for-helm";
          version = "2.6.1";

          src = pkgs.fetchFromGitHub {
            owner = "bitnami";
            repo = "readme-generator-for-helm";
            rev = "2.6.1";
            hash = "sha256-hgVSiYOM33MMxVlt36aEc0uBWIG/OS0l7X7ZYNESO6A=";
          };

          npmDepsHash = "sha256-baRBchp4dBruLg0DoGq7GsgqXkI/mBBDowtAljC2Ckk=";
          dontNpmBuild = true;
        };

        kubeApiLinterPlugin = pkgs.buildGoModule {
          pname = "kube-api-linter-plugin";
          version = "0-unstable-2026-02-19";

          src = pkgs.fetchFromGitHub {
            owner = "jonstacks";
            repo = "kube-api-linter";
            rev = "4d8081c836df2bcaf5ae0d173ddd9e7b1f9c2c37";
            hash = "sha256-LTh42PzmVrCy05wphy1fV+8auO20+XgG03052TTuSu0=";
          };

          vendorHash = "sha256-RHfx8RWbuZz8iPUXFK19/FY1FtesuP3aLH6a3bKOu9I=";

          env.CGO_ENABLED = "1";

          buildPhase = ''
            runHook preBuild
            go build -buildmode=plugin -o kube-api-linter.so ./pkg/plugin
            runHook postBuild
          '';

          installPhase = ''
            runHook preInstall
            mkdir -p $out/lib
            cp kube-api-linter.so $out/lib/
            runHook postInstall
          '';

          doCheck = false;
        };

        # golangci-lint with CGO enabled for loading .so plugins
        golangci-lint-cgo = pkgs.golangci-lint.overrideAttrs (old: {
          env = (old.env or { }) // {
            CGO_ENABLED = "1";
          };
          dontStrip = true;
        });

        mkScript =
          name: text:
          let
            script = pkgs.writeShellScriptBin name text;
          in
          script;

        scripts = [
          (mkScript "devhelp" ''
            cat <<'EOF'

            Welcome to the ngrok-operator development environment!

            Please make sure you have the following environment variables set:

              NGROK_API_KEY      - Your ngrok API key
              NGROK_AUTHTOKEN    - Your ngrok authtoken

            If you are using GitHub Codespaces, a kind cluster should
            already be running. You can verify this by running:

              kind get clusters

            Common commands:
              make build          - Build the operator
              make test           - Run tests
              make lint           - Run linters
              make deploy         - Deploy to the kind cluster

            For more information, see the development documentation in

              ./docs/developer-guide/README.md

            You can also run "devhelp" at any time to see this message again.
            EOF
          '')
        ];
      in
      {
        packages.readme-generator-for-helm = readmeGeneratorForHelm;
        packages.kube-api-linter-plugin = kubeApiLinterPlugin;

        devShells.default = pkgs.mkShell {
          buildInputs =
            with pkgs;
            [
              bashInteractive
              go_1_26
              go-tools
              golangci-lint-cgo
              gotools
              jq
              kind
              kubebuilder
              kubectl
              kubernetes-controller-tools
              (pkgs.wrapHelm pkgs.kubernetes-helm {
                plugins = [
                  pkgs.kubernetes-helmPlugins.helm-unittest
                ];
              })
              kyverno-chainsaw
              ngrok
              nixfmt-rfc-style
              setup-envtest
              tilt
              yq
              readmeGeneratorForHelm
            ]
            ++ scripts;

          CGO_ENABLED = "0";
          # GitHub Codespaces sets GOROOT in /etc/environment. However, we are managing
          # go via nix, so we need to unset it to avoid conflicts. See also: https://dave.cheney.net/2013/06/14/you-dont-need-to-set-goroot-really
          GOROOT = "";

          ENVTEST_K8S_VERSION = "1.34.1";

          shellHook = ''
            mkdir -p bin
            ln -sf ${kubeApiLinterPlugin}/lib/kube-api-linter.so bin/kube-api-linter.so
            export KUBEBUILDER_ASSETS="$(setup-envtest use $ENVTEST_K8S_VERSION -p path)"
            devhelp
          '';
        };
      }
    ));
}
