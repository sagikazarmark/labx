{
  pkgs,
  config,
  ...
}:

let
  labctl = pkgs.buildGoModule.override { go = config.languages.go.package; } (finalAttrs: {
    pname = "labctl";
    version = "0.1.116";

    src = pkgs.fetchFromGitHub {
      owner = "iximiuz";
      repo = "labctl";
      tag = "v${finalAttrs.version}";
      hash = "sha256-a2GUNR8lhAtvE15c5xv4VxEP9eckru9AHYiiCvBNukI=";
    };

    vendorHash = "sha256-YNjguFRCgm3W5fsyUXRPXka0sWJUJYCXMhv9tAL+JYU=";

    subPackages = [ "." ];

    ldflags = [
      "-s"
      "-w"
      "-X main.version=${finalAttrs.version}"
      "-X main.commit=v${finalAttrs.version}"
      "-X main.date=1970-01-01T00:00:00Z"
    ];

    doCheck = false;
  });
in
{
  packages = [
    pkgs.just
    pkgs.golangci-lint
    pkgs.goreleaser
    labctl
  ];

  languages = {
    go = {
      enable = true;
      version = "1.27.4";
    };
  };
}
