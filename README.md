# Table of Content

   * [Simple builder](#simple-builder)
   * [Installation](#installation)
   * [Recommended .envrc](#recommended-envrc)
   * [Available branches](#available-branches)
      * [Nomad job](#nomad-job)
      * [Configuration](#configuration)
      * [Behaviour](#behaviour)
      * [Releasing simple-builder](#releasing-simple-builder)
      * [Example job configuration](#example-job-configuration)

# Simple builder

Simple builder makes it possible to clone a git repository locally, build it
and then push the build logs to configured callbacks.

# Installation

Just clone this repo in your `$GOPATH/src/github.com/squarescale/simple-builder`

# Recommended .envrc

If you are a [direnv](https://direnv.net/) user here is a recommended `.envrc`

```sh
export GIT_COMMITTER_NAME="You"
export GIT_COMMITTER_EMAIL="you@squarescale.com"

export GIT_AUTHOR_NAME="You"
export GIT_AUTHOR_EMAIL="you@squarescale.com"

export GO111MODULE="on"

export GITHUB_USER_TOKEN=you:yourgithubapitoken
```

## Nomad job

Unlike many other jobs `simple-builder` is not a traditional Nomad service.
It is a parameterized job. Everytime a user triggers a private build from the
UI [squarescale-web] triggers a parameterized job in Nomad as declared in
[app/models/repository.rb](https://github.com/squarescale/squarescale-web/blob/env-production/app/models/repository.rb#L102-L122).

The nomad job definition is automatically generated in [squarescale-web] in
[app/lib/nomad/simple_builder_job.rb](https://github.com/squarescale/squarescale-web/blob/env-production/app/lib/nomad/simple_builder_job.rb).

## Configuration

The following environnment variables are provided in the nomad job definition
but these are not currently used in `simple-builder` as there is no need to.

Name | Usage
-----|------
`SQSC_PROJECT` | Project name
`SQSC_PROJECT_UUID` | Project UUID
`SQSC_ENVIRONMENT` | AWS environment (`production`, `stating`, etc)


## Behaviour

For every invocation, `simple-builder` will execute the following tasks:

1. `git clone` the repository provided in `git_url` using `git_secret_key`
2. Generate a script file containing `build_script` and execute it
3. Send the entire build log to `callbacks` one the script is executed

## Releasing simple-builder

Given you have configured `GITHUB_USER_TOKEN` as described above you can simply
run `make publish`.

## Example job configuration

```json
    {
      "callbacks": [
        "https://www.example.com/foo/bar/baz"
      ],

      "git_url": "git@github.com:squarescale/sqsc-demo-app.git",

      "git_secret_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBA[...]yp71g==\n-----END RSA PRIVATE KEY-----\n",

      "build_script": "
      #!/bin/bash
      set -e
      cp -r /root/.docker $HOME/.docker
      docker build --no-cache --memory-swap -1 -t xxx/project-1234-5678:v1 .
      docker push xxx/project-1234-5678:v1
      "
    }
```

[squarescale-web]: (https://github.com/squarescale/squarescale-web)
