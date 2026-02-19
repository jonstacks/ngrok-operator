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

        golangciLintKubeApiLinter = pkgs.buildGoModule {
          pname = "golangci-lint-kube-api-linter";
          version = "0-unstable-2026-02-06";

          src = pkgs.fetchFromGitHub {
            owner = "kubernetes-sigs";
            repo = "kube-api-linter";
            rev = "39e3d06a2850e38a8e9e82918bab14ce84e608de";
            hash = "sha256-P7Gy8JeOxiCvy0TW9IPz9cDc/20gD9QJNEO5Tao10JY=";
          };

          vendorHash = "sha256-ITaN1Ge/SVRQevmbvj9vcBE6rAPqHyydU+RNlryC1Eg=";

          subPackages = [ "cmd/golangci-lint-kube-api-linter" ];

          # Provide golangci-lint symlink so existing make targets work unchanged.
          postInstall = ''
            ln -s $out/bin/golangci-lint-kube-api-linter $out/bin/golangci-lint
          '';

          doCheck = false;
        };

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
        packages.golangci-lint-kube-api-linter = golangciLintKubeApiLinter;

        devShells.default = pkgs.mkShell {
          buildInputs =
            with pkgs;
            [
              go
              go-tools
              golangciLintKubeApiLinter
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
            export KUBEBUILDER_ASSETS="$(setup-envtest use $ENVTEST_K8S_VERSION -p path)"
            devhelp
          '';
        };
      }
    ));
}
