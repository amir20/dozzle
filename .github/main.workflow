workflow "Build, Test and Release" {
  on = "push"
  resolves = [
    "Release",
  ]
}

action "go test" {
  uses = "./.github/golang/"
}

action "npm test" {
  uses = "actions/npm@master"
  args = "it"
}

action "Tag" {
  uses = "actions/bin/filter@master"
  needs = ["go test", "npm test"]
  args = "tag"
}

action "Release" {
  uses = "./.github/goreleaser/"
  needs = ["Tag"]
  args = "release"
  secrets = ["GITHUB_TOKEN", "DOCKER_USERNAME", "DOCKER_PASSWORD"]
}
