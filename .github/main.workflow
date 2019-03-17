workflow "Release" {
  on = "push"
  resolves = [
    "goreleaser/goreleaser",
  ]
}

action "cedrickring/golang-action@1.2.0" {
  uses = "cedrickring/golang-action@1.2.0"
  args = "go test ./..."
}

action "actions/bin/filter@master" {
  uses = "actions/bin/filter@master"
  needs = ["cedrickring/golang-action@1.2.0"]
  args = "tag"
}

action "goreleaser/goreleaser" {
  uses = "docker://goreleaser/goreleaser"
  needs = ["actions/bin/filter@master"]
  args = "release"
}
