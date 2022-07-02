# User <-> GitHub <-> Bass Loop sequence diagram

The following diagram shows how all the components interact in a typical flow
using Bass Loop to run thunks published as status checks to GitHub.

It's pretty involved and might have too much detail; PRs welcome for clarity!

The example user 'vito' below may be a project maintainer or a contributor.

There are additional notes included in the source markdown file that explain
even more details; they're commented out so it's not too overwhelming.

```mermaid
sequenceDiagram
    autonumber
    actor Developer as vito
    participant Runner as bass --runner
    participant SSH as github.bass-lang.org
    participant GitHub
    participant DB as SQLite: loop.db
    participant HTTP as loop.bass-lang.org
    participant Project as project.bass
    participant Blobs as Loop Blobstore
    Developer->>Runner: start bass runner
    note over Developer: bass --runner vito@github.bass-lang.org
    Runner->>+SSH: forward runtimes over SSH
    SSH->>GitHub: get user and verify key
    %% Note over GitHub: GET /users/vito
    %% Note over GitHub: GET /users/vito/keys
    SSH->>DB: store user's runtimes
    %% Note over DB: INSERT INTO runtimes SET user_id = github_user_id, name = session_id
    %% loop every minute
    %%     Runner-->>SSH: keepalive
    %% end
    loop each event
    Developer->>GitHub: cause webhook event
    Note over Developer: git push
    GitHub->>HTTP: send webhook
    %% Note over HTTP: POST /integrations/github/events
    HTTP->>DB: fetch event sender's runtimes
    %% Note over DB: SELECT * FROM runtimes WHERE user = event_sender
    HTTP->>GitHub: load project.bass from event repo
    %% Note over GitHub: GET /repos/vito/bass/contents/project.bass
    HTTP->>Project: call github-hook
    loop each check
        Project->>DB: create thunk run
        %% Note over DB: INSERT INTO runs SET thunk_id = thunk
        Project->>GitHub: create check run
        %% GitHub-->>Developer: github status checks show up
        %% Note over GitHub: POST /repos/vito/bass/check-runs
        Project->>+Runner: run thunk
        %% Note over Runner: gRPC Run(Thunk)
        Runner--)Project: stream vertex data/logs
        %% Note over Runner: returns (stream RunResponse)
        Runner-->>-Project: run completes
        Project->>DB: update thunk run status
        %% GitHub-->>Developer: github status checks update
        Project->>DB: store vertex metadata
        %% Note over DB: INSERT INTO vertexes SET name = "git clone ...
        Project->>Blobs: store vertex logs
        %% Note over Blobs: pre-rendered HTML
        Project->>GitHub: update check run status
    end
    end
    break runner disconnects
        SSH->>-DB: clean up runtimes
        %% Note over DB: DELETE FROM runtimes WHERE name = session_id
        Runner-)SSH: reconnect with backoff
    end
    Developer->>GitHub: user views checks
    Note over Developer: click check statuses on commit/PR
    Developer->>HTTP: user views build output
    Note over Developer: click GitHub check Details link
    HTTP->>Blobs: fetch vertex logs
    HTTP-->>Developer: render run output
```
