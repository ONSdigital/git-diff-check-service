Git diff check service
======================

A webservice for watching git push hooks and checking the diffs for
potentially sensitive data

Uses the [git-diff-check](https://github.com/ONSdigital/git-diff-check) library

## Getting started

The following will set this service up for a whole organization. You can do
similar if you wish on individual repositories.

- In github, go to your `https://github.com/organizations/<YOUR_ORG>/settings/hooks`
- Click `Add Webhook`
- Enter the required info:
    - The url where the service is/will be running. e.g. `https://<your_url>/push`
    - Set the content type to `application/json`
    - Create a `webhook secret` of sufficient entropy
    - Leave `Just the push event` selected and click `Add webhook`

The service requires the following environment variables to be set:

| Env var        | Example           | Description |
|----------------|-------------------|-------------|
| PORT           | `5000`            | The port number to run the service on |
| WEBHOOK_SECRET | `thisIsABadSecret`| The secret key that was entered when setting up the webhook above |

## API

The service provides the following endpoints:

- `/push` (`POST`) Receives a _push event_ webhook from github.
  - Expects `Content-Type: application/json`
  - Expects (https://developer.github.com/v3/activity/events/types/#pushevent) payload.
  - Expects `X-Hub-Signature` containing message signature (payload signed with `WEBHOOK_SECRET`)
  - Returns `200 OK` if ok
  - Returns an api problem report and appropriate status code if an error occurs

License
=======

Copyright (c) 2017 Crown Copyright (Office for National Statistics)

Released under MIT license, see [LICENSE](LICENSE) for details.