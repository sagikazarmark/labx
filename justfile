[private]
default:
  @just --list

test:
  @go test -v ./labx -root-path $PWD/../iximiuz-labs-content
