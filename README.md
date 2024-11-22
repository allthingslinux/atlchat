# atlchat is modern IRC server written in Go.

**This project is a fork of [Ergo](https://github.com/ergochat/ergo)**   

Extra features in atlchat:
* Automod, which lets you block messages using customizable rules with word lists or regex.
* Customizable chat cooldowns, much like Discord's slowmode feature.

## Self-hosting / developing
This project is a stripped down version configured to to work specifically for `irc.atl.chat`. You can run this project yourself with Docker or baremetal:

1. Clone this repository
2. Copy `default.yaml` to `ircd.yaml` and configure the server ([see Ergo's docs](https://github.com/ergochat/ergo/tree/master/docs))
3. Make sure to create a file called `ergo.motd` before starting the server
4. `docker compose up atlchat` or `make build` to create an `ergo` binary. See `./ergo help` for more info.

Note: `matterbridge` is in the compose file in case you want to configure it, this will not work out of the box.


## Credits

* Jeremy Latt (2012-2014)
* Edmund Huber (2014-2015)
* Daniel Oaks (2016-present)
* Shivaram Lingamneni (2017-present)
* [Many other contributors and friends of the project <3](https://github.com/ergochat/ergo/blob/master/CHANGELOG.md)
