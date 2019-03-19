workflow "Release" {
  on = "push"
  resolves = [
    "release",
  ]
}

action "test" {
  uses = "./.github/golang/"
}

action "is-tag" {
  uses = "actions/bin/filter@master"
  needs = ["test"]
  args = "tag"
}

action "release" {
  uses = "./.github/goreleaser/"
  needs = ["is-tag"]
  args = "release"
  secrets = ["GITHUB_TOKEN", "DOCKER_USERNAME", "DOCKER_PASSWORD"]
}
