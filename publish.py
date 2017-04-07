#!/usr/bin/python3

import os, requests, subprocess, sys

user_token = os.environ.get("GITHUB_USER_TOKEN")
if user_token is None:
    print("GITHUB_USER_TOKEN not defined.")
    sys.exit(1)

github_repo = os.environ.get("GITHUB_REPO")
if github_repo is None:
    print("GITHUB_REPO not defined.")
    sys.exit(1)

bin_files = os.environ.get("BIN_FILES")
if bin_files is None:
    print("BIN_FILES not defined.")
    sys.exit(1)
else:
    bin_files = bin_files.split()

user  = user_token.split(":")[0]
token = user_token.split(":")[1]

# Get all releases
print("Retrieve all releases...")
url = "https://api.github.com/repos/%s/releases" % github_repo
releases = requests.get(url, auth=(user, token))
if releases.status_code != 200:
    print("Error: GET releases " + str(releases.status_code))
    sys.exit(1)

# Clear all previous drafts
print("Remove old release drafts...")
for release in releases.json():
    if release["draft"]:
        delete_url = url + "/" + str(release["id"])
        deleted = requests.delete(delete_url, auth=(user, token))
        if deleted.status_code != 204:
            print("Error: DELETE draft " + str(deleted.status_code))
            sys.exit(1)

# Find sha1
git_sha_1_process = subprocess.run(
    "git describe --always", stdout=subprocess.PIPE,
    shell=True, check=True, universal_newlines=True)
if git_sha_1_process.returncode != 0:
    print("Error: Cannot retrieve sha1 of this repository")
    sys.exit(1)

git_sha_1 = git_sha_1_process.stdout.replace("\n", "")

# Create new release
print("Create new release draft...")
headers = { "Content-Type": "application/json" }
data = {
    "tag_name": git_sha_1,
    "name": "release " + git_sha_1,
    "draft": False
}

created = requests.post(url, auth=(user, token), headers=headers, json=data)
if created.status_code != 201:
    print("Error: POST draft " + str(created.status_code))
    sys.exit(1)

release_id = str(created.json()["id"])

# Push executable contents
for executable in bin_files:
    print("Push executable " + executable + " to release draft...")
    params = { "name": executable }
    headers = { "Content-Type": "application/octet-stream" }
    upload_url = "https://uploads.github.com/repos/%s/releases/%s/assets" % \
            (github_repo, release_id)

    with open(executable, "rb") as f:
        data = f.read()

        uploaded = requests.post(
            upload_url, auth=(user, token),
            headers=headers, params=params, data=data)

        if uploaded.status_code != 201:
            print("Error: Upload " + executable + " " + str(uploaded.status_code))
            sys.exit(1)

print("Done.")
