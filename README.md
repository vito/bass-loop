# bass loop

A continuous [Bass](https://github.com/vito/bass) service. Currently geared towards GitHub but other integrations should be possible.

See [the Announcement](https://github.com/vito/bass-loop/discussions/1) for more details - a proper README will come shortly!

## demo

See [Bass Loop demo](https://github.com/vito/bass-loop-demo) for a repo to play
around with.

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
