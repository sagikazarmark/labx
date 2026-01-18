{
  pkgs,
  ...
}:

{
  packages = [
    pkgs.just
    pkgs.golangci-lint
    pkgs.goreleaser
  ];

  languages = {
    go = {
      enable = true;
    };
  };
}
