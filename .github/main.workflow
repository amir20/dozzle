workflow "Release" {
  on = "push"
  resolves = [
    "goreleaser/goreleaser",
  ]
}

action "go-build" {
  uses = "./.github/golang/"
}

action "is-tag" {
  uses = "actions/bin/filter@master"
  needs = ["go-build"]
  args = "tag"
}

action "goreleaser/goreleaser" {
  uses = "docker://goreleaser/goreleaser"
  needs = ["is-tag"]
  args = "release"
}
