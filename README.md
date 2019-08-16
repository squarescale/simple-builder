simple-builder
==============

Simple-builder is a simple daemon that clones git repositories and execute build
commands on it.

The build commands are supposed to be trusted, there is no isolation of the
executed commands.

Building
--------

Just run `make gobuild`. Run `make help` for help.

Releasing
---------

You must create a tag for your release, like `git tag -a vX.Y.Z`.

Then you can run:

    GITHUB_USER_TOKEN=<github_username>:<github_rest_api_token> make publish
