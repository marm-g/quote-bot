# quoteBot - a simple way to store and retrieve quotes via Discord

## Setup

*Disclaimer: the bot isn't really made to be modular/support anyone's uses besides mine at the moment. Use with caution!*

First, set up your SQLite database. The schema is provided in `schema.sql`, and the app expects
the database to be located inside the srcdir, named `quotes.db`. This can be changed in the code,
if you want.

Next, run `go build`, then execute `quote-bot`. The bot expects a OAuth Token to be provided
via the `QB_OAUTH_TOKEN` environment variable.

## Sample systemd service file

```
[Unit]
Description=Quote bot for Discord.
Requires=network.target
After=network.target

[Service]
Type=simple
Restart=always
RestartSec=10
User=quotebot
Group=bots
ExecStart=/<path to bot>/quote-bot/quote-bot
WorkingDirectory=/<path to bot>/quote-bot/
Environment="QB_OAUTH_TOKEN=<OAuth Bot Token here>"

[Install]
WantedBy=multi-user.target
````
