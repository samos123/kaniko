# Development

This doc explains the development workflow so you can get started
[contributing](CONTRIBUTING.md) to Kaniko!

## Getting started

First you will need to setup your GitHub account and create a fork:

1. Create [a GitHub account](https://github.com/join)
1. Setup [GitHub access via
   SSH](https://help.github.com/articles/connecting-to-github-with-ssh/)
1. [Create and checkout a repo fork](#checkout-your-fork)

Once you have those, you can iterate on kaniko:

1. [Run your instance of kaniko](README.md#running-kaniko)
1. [Verifying kaniko builds](#verifying-kaniko-builds)
1. [Run kaniko tests](#testing-kaniko)

When you're ready, you can [create a PR](#creating-a-pr)!

## Checkout your fork

The Go tools require that you clone the repository to the `src/github.com/GoogleContainerTools/kaniko` directory
in your [`GOPATH`](https://github.com/golang/go/wiki/SettingGOPATH).

To check out this repository:

1. Create your own [fork of this
  repo](https://help.github.com/articles/fork-a-repo/)
2. Clone it to your machine:

  ```shell
  mkdir -p ${GOPATH}/src/github.com/GoogleContainerTools
  cd ${GOPATH}/src/github.com/GoogleContainerTools
  git clone git@github.com:${YOUR_GITHUB_USERNAME}/kaniko.git
  cd kaniko
  git remote add upstream git@github.com:GoogleContainerTools/kaniko.git
  git remote set-url --push upstream no_push
  ```

_Adding the `upstream` remote sets you up nicely for regularly [syncing your
fork](https://help.github.com/articles/syncing-a-fork/)._

## Verifying kaniko builds

Images built with kaniko should be no different from images built elsewhere.
While you iterate on kaniko, you can verify images built with kaniko by:

1. Build the image using another system, such as `docker build`
2. Use [`container-diff`](https://github.com/GoogleContainerTools/container-diff) to diff the images

## Testing kaniko

kaniko has both [unit tests](#unit-tests) and [integration tests](#integration-tests).

### Unit Tests

The unit tests live with the code they test and can be run with:

```shell
make test
```

_These tests will not run correctly unless you have [checked out your fork into your `$GOPATH`](#checkout-your-fork)._

### Integration tests

The integration tests live in [`integration`](./integration) and can be run with:

```shell
make integration-test
```

_These tests require push access to a project in GCP, and so can only be run
by maintainers who have access. These tests will be kicked off by [reviewers](#reviews)
for submitted PRs._

## Creating a PR

When you have changes you would like to propose to kaniko, you will need to:

1. Ensure the commit message(s) describe what issue you are fixing and how you are fixing it
   (include references to [issue numbers](https://help.github.com/articles/closing-issues-using-keywords/)
   if appropriate)
1. [Create a pull request](https://help.github.com/articles/creating-a-pull-request-from-a-fork/)

### Reviews

Each PR must be reviewed by a maintainer. This maintainer will add the `kokoro:run` label
to a PR to kick of [the integration tests](#integration-tests), which must pass for the PR
to be submitted.