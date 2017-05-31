simple-builder
==============

Simple-builder is a simple daemon that clones git repositories and execute build
commands on it. It takes the build orders via an HTTP API and allows you to be
notified either by watching the build via an HTTP request, or via an HTTP
callback.

The build commands are supposed to be trusted, there is no isolation of the
executed commands.

Build
-----

Just run `make`. Run `make help` for help.


Execute
-------

Just run the executable. You can configure the listening port using the
`NOMAD_ADDR_http` or via the command line flag `--http`.

API
---

- `GET /version`: reply the version number

- `GET /health`:  reply that the service is healthy

- `POST /builds`: schedule a new build. Takes a JSON object with:

    - `callbacks`: list of strings to call back with the job status one finished
    - `build_script`: script to execute for the build (current directory is the
      checkout directory)
    - `git_url`: the Git checkout URL
    - `git_secret_key`: optional SSH secret key for the git clone
    - `git_branch`: optional git branch to clone
    - `git_full_clone`: don't limit depth for git clone
    - `git_recursive`: clone recursively
    - `git_checkout_dir`: subdirectory name to clone to, defaults to the last
      part of the `git_url`

    Respond with a `Location:` header that points to `/builds/<id>`

    Callbacks are passed as request body the same information that
    `GET /builds/<id>` already returns.

- `GET /builds/<id>`: Get as JSON the status of the build. The JSON object
  contains the build parameter attributes and additionally:

    - `process_state`: an object containing the last process state that exited
    - `errors`: a list of strings containing the build errors
    - `output`: a string containing the build output

- `GET /builds/<id>/wait`: Same as `/builds/<id>` but only return when the build
  is finished

- `GET /builds/<id>/output`: Plain text output of the build
