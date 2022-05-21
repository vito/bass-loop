# bass loop

A continuous [Bass](https://github.com/vito) service. Currently geared towards GitHub but other integrations should be possible.

## the plan

* [x] A GitHub app for running Bass GitHub event handlers in-repo (kinda like GitHub actions).
    * [ ] A shorthand for the common case of running checks.
* [ ] A web UI for viewing thunk output (so a 'details URL' can be set on GitHub checks).
    * [ ] A thunk that contains secrets should default to private visibility.
* [ ] A SSH server so that users can bring their own workers (i.e. their local machine).
    * [ ] A method for passing secrets to thunks via the worker so sensitive values never even leave the machine.
    * [ ] A method for PR authors to satisfy PR checks using their own workers, without the repo maintainer having to run them.
* [ ] Scalable - everyone brings-their-own-worker, so only the Loop has to be scaled out.
