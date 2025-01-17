## About

![Docker](https://github.com/pankajsoni19/listmonk/actions/workflows/build.yml/badge.svg)

This is a forked repo from `v4.1.0` upstream. It maintains compatibility with upstream. It contains these additional work items. Changes are extensive, so most probably wouldnt get merged.

Visit [listmonk.app](https://listmonk.app) for more info.

#### Changes from upstream

I tagged it to db schema version, and will follow that as semver. Current `5.3.0`

##### Long running campaigns

- When creating campaign choose run type as `Event Subscription`
- Supported via
  - Modify subscriber list memberships -> `/api/subscribers/lists`
  - Create a new subscriber -> `/api/subscribers`
- If no new message, worker will sleep for 1 minute before querying

##### Weighted From

- From is moved into smtp, messenger config.
- Sample -> `One <one@example.com>,10,Two <two@value.com>,5`. This will send based on weights assigned. Can be phone number for messenger.

##### Per Campaign smtp/messenger

- In `Settings`>`SMTP` config specify a name.
- In create/update campaign specify same name.
- It will use that messenger to send outbound event.
- Specify a `default`, if none specified picks first. It is used to send system alert emails.

##### Query list by exact name

- `/api/lists` specify `?name=` for exact lookup. Specify multiple times to search multiple

##### Per Campaign `Sliding window limit`

- Config added to create/update campaign UI
- Config is removed from `Settings`>`Performance`. On upgrade this is copied to all campaigns.
- Allows setting limit per campaign per endpoint.

##### Cloning of lists

- Cloning action button on lists page. It copies all config, subscribers to the new list.

##### Bug fixes

- Campaign pause/stop.
  - Though stopped on UI, it used to run in bg over the last fetched subscriber list and loop stops quite late.
  - Server config change forces a restart so the delta of unprocessed subscribers get lost.

##### Default data

- Skips default data creation for list, template, campaigns on install.
- check_updates is false

## Installation

Assumes we are on debian

### Via docker

```bash
docker pull pankaj19soni/listmonk:latest

# Download the compose file to the current directory.
# Update image name with tag created above
curl -LO https://raw.githubusercontent.com/pankajsoni19/listmonk/master/docker-compose.yml

# Run the services in the background.
docker compose up -d
```

## Build

### Requirements

- yarn 2
- nodejs 20
- golang latest
- postgres >= 12

### Build container

```bash

apt-get install -yq build-essential automake autoconf pkg-config cmake libssl-dev git git-lfs

cd dependencies
curl -sL https://deb.nodesource.com/setup_20.x | sudo -E bash -
apt update
apt install nodejs
node -v

corepack enable
yarn set version stable
yarn install

wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
sudo export PATH=$PATH:/usr/local/go/bin

cd ..
git clone https://github.com/pankajsoni19/listmonk

cd listmonk

# This builds the binary
make dist

# This wraps the binary into a docker image
docker build -t YOUR-TAG .
```

## Upgrade

- Take db backup
- Stop previous container/binary.
- Update compose file to point to new image or point to new binary and start

Visit `http://localhost:9000`

---

### Binary

- `./listmonk --new-config` to generate config.toml. Edit it.
- `./listmonk --install` to setup the Postgres DB (or `--upgrade` to upgrade an existing DB. Upgrades are idempotent and running them multiple times have no side effects).
- Run `./listmonk` and visit `http://localhost:9000`

See [installation docs](https://listmonk.app/docs/installation)

---

## Developers

listmonk is free and open source software licensed under AGPLv3. If you are interested in contributing, refer to the [developer setup](https://listmonk.app/docs/developer-setup). The backend is written in Go and the frontend is Vue with Buefy for UI.

## License

listmonk is licensed under the AGPL v3 license.
