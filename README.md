# Lil' Dumpster Bot

URL: https://ptb.discord.com/api/oauth2/authorize?client_id=784836525060784178&permissions=2415994880&scope=bot%20applications.commands

Permissions:
* General - Manage Roles
* Text - Send Messages
* Text - Manage Messages
* Text - Read Message History
* Text - Use Slash Commands

## Testing

This bot may be tested locally provided that you have a DISCORD_TOKEN environment variable set to a valid Discord token. You may manage your own token by creating an App at https://ptb.discord.com/developers/applications which will allow you to develop new features independently of the hosted bot (e.g. "production").

```sh
export DISCORD_TOKEN=xxxtokenxxx
go run main.go
```

Alternatively if you need to develop against the bot directly, coordinate with the repository owner(s) and we can shutdown the existing bot and distribute its token to you.
