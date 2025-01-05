## About

This is a forked repo from `v4.1.0` upstream. It maintains compatibility with upstream. It contains these additional work items. Changes are extensive, so most probably wouldnt get merged.

Visit [listmonk.app](https://listmonk.app) for more info.

#### Requirements

* yarn 2
* nodejs 20
* golang latest
* postgres >= 12

#### Changes from upstream

I tagged it to db migration version, and will follow that as semver.

##### Long running campaigns
* when creating campaign choose run type as `Event Subscription`
* Modify subscriber list memberships -> `/api/subscribers/lists`
* Create a new subscriber -> `/api/subscribers`
* If no new message, worker will sleep for 1 minute before querying

##### Per Campaign smtp/messenger

* In smtp config specify a name.
* In create/update campaign specify same name. 
* It will use that messenger to send outbound event.

##### Query list by exact name

* `/api/lists` specify `?name=` for exact lookup. Specify multiple times to search multiple

##### Per Campaign `Sliding window limit`

* config added to create/update campaign UI
* config is removed from `Settings`>`Performance`
* Allows setting limit per campaign per endpoint.

##### Cloning of lists

* Cloning action button on lists page. It copies all config, subscribers to the new list.

##### Bug fixes

* Campaign pause/stop. 
	* Though stopped on UI, it used to run in bg over the last fetched subscriber list and loop stops quite late.
	* Server config change forces a restart so the delta of unprocessed subscribers get lost.

##### Default data

* Skips default data creation for list, template, campaigns on install.
* check_updates is false

## Installation

Assumes we are on debian

```bash

apt-get install git git-lfs

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

cd ..

# Download the compose file to the current directory.
# Update image name with tag created above
curl -LO https://raw.githubusercontent.com/pankajsoni19/listmonk/master/docker-compose.yml

# Run the services in the background.
docker compose up -d
```

For upgrade build the docker/binary. 

* Stop previous container/binary.
* Update compose file to point to new image or point to new binary and start

Visit `http://localhost:9000`

__________________

### Binary
- `./listmonk --new-config` to generate config.toml. Edit it.
- `./listmonk --install` to setup the Postgres DB (or `--upgrade` to upgrade an existing DB. Upgrades are idempotent and running them multiple times have no side effects).
- Run `./listmonk` and visit `http://localhost:9000`

See [installation docs](https://listmonk.app/docs/installation)
__________________


## Developers
listmonk is free and open source software licensed under AGPLv3. If you are interested in contributing, refer to the [developer setup](https://listmonk.app/docs/developer-setup). The backend is written in Go and the frontend is Vue with Buefy for UI. 


## License
listmonk is licensed under the AGPL v3 license.
