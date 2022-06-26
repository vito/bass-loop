# bass loop

A continuous [Bass](https://github.com/vito/bass) service. Currently geared towards GitHub but other integrations should be possible.

See [the Announcement](https://github.com/vito/bass-loop/discussions/1) for more details - a proper README will come shortly!

## demo

See [Bass Loop demo](https://github.com/vito/bass-loop-demo) for a repo to play
around with.

## installation

Note: you only need to install Bass Loop if you're planning to run your own
server.

Using Go 1.18+:

```sh
go install github.com/vito/bass-loop/cmd/bass-loop@latest
```

## the plan

* [x] A GitHub app for running Bass GitHub event handlers in-repo (kinda like GitHub actions).
    * [x] A shorthand for the common case of running checks.
* [x] A web UI for viewing thunk output (so a 'details URL' can be set on GitHub checks).
    * [ ] A thunk that contains secrets should default to private visibility.
* [x] A SSH server so that users can bring their own workers (i.e. their local machine).
    * [ ] A method for passing secrets to thunks via the runner so sensitive values never even leave the machine.
    * [x] A method for PR authors to satisfy PR checks using their own workers, without the repo maintainer having to run them.
* [ ] Scalable - everyone brings-their-own-worker, so only the Loop has to be scaled out.
* [ ] Make it a little more friendly. Right now the frontpage is pretty cryptic; it's purely driven by the 'navigating from GitHub' use case at the moment, but a dash of metadata could help tie things back in the other direction.

## GitHub App configuration

First, go to [Register new GitHub App](https://github.com/settings/apps/new).
This guide will walk you through the creation steps. Don't worry too much as
everything can be changed later.

Set the **GitHub App name** to whatever you want, but keep in mind this is a
global (to GitHub) namespace. Set whatever description you deem appropriate -
this will show up when users view your app. Feel free to steal the description
from my [Bass CI app](https://github.com/apps/bass-ci).

Set the **Homepage URL** to the external URL of your app e.g.
`https://example.com`. If you're kicking the tires, use
[ngrok](https://ngrok.com/) to serve your local Loop to the public internet:

use an address like `https://abcd-123-45-67-89.ngrok.io`.

Skip the "Identifying and authorizing users" section - it's unused.

Skip the "Post installation" section too unless you've got your own page to
take them to. Loop might provide one of these someday; it'd be nice UX for
getting started.

### Webhook

Enable webhooks.

Set **Webhook URL** to e.g. `https://example.com/integrations/github/events`

Set a **Webhook secret** for a real public installation so people can't spoof
webhook payloads.

### Repository permissions

**Checks**: Read and write. This is Loop's main function.

**Contents**: Read-only. Needed to receive `push` events. Also needed to fetch
the repo's `project.bass` script.

The remaining permissions are up to you; it depends on what type of events you
want to send to repos. As you enable more permissions, more events become
available in the next "Subscribe to events" section later on.

**Pull requests**: Read-only. To be honest, this might not be necessary.

### Organization permissions

No access required.

### User permissions

No access required.

### Subscribe to events

As stated before this is up to you, but here are the events enabled for
[loop.bass-lang.org](https://loop.bass-lang.org) at the time of writing:

* [x] Meta
* [x] Pull request
* [x] Push
* [x] Release

### Create the app

When it asks **Where can this GitHub App be installed?**, pick which option
makes sense for you. You can always change it later, so it's probably safest to
leave it as **Only on this account** since it's harder to restrict it once
people have installed it.

Finally, click **Create GitHub App**!

### Display Information

Once you've created your app you can set a nice logo! This will show up on
status checks and on the app itself.

You can generate a Bass logo in your favorite colors like so:

```sh
git clone https://github.com/vito/bass
cd bass/
./docs/scripts/generate-logo docs/ico/base16-<THEME>.svg logo.png
```

### Private keys

Your app needs a private key configured in order to make requests on behalf of
users. Click "Generate a private key" to create and download one.

### Run `bass-loop` with GitHub app config

Copy your app ID from the top of the settings page, put your private key
somewhere within reach, and run:

```sh
bass-loop \
  --github-app-id 12345 \
  --github-app-key app-private-key.pem \
  --github-app-webhook-secret mysecret
```

These parameters can also be set using env vars. I personally use 1Password and
the `op` CLI like so:

```sh
cat > creds.env <<EOF
GITHUB_WEBHOOK_SECRET = op://Bass/webhook-secret/password
GITHUB_APP_PRIVATE_KEY = op://Bass/github-app-private-key/private-key
EOF

eval $(op signin)

op run --no-masking --env-file ./creds.env bass-loop
```
