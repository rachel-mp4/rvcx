# rvcx
this is the backend for the xcvr appview. it uses lrcd to host the lrc servers,
postgres to cache the atproto data, atproto-oauth-golang as an oauth client,
and probably some other things i am forgetting!

## principles
the xcvr atproto ecosystem has backend appviews which are simultaneously 
atproto actors who write signet records to their repos in order to have 
bidirectional confidence that off-protocol communication occured

## how to run
i don't really test this in development since i added oauth, this likely will 
change in the future

in production, you must create a .env file that contains the following
environment variables, setting secrets as appropriate

```
POSTGRES_USER=xcvr
POSTGRES_PASSWORD=secret
POSTGRES_DB=xcvrdb
POSTGRES_PORT=15432
MY_NAME=xcvr
MY_IDENTITY=xcvr.org
MY_SECRET=secret
MY_METADATA_PATH=/meta/client-metadata.json
MY_TOS_PATH=/meta/tos
MY_POLICY_PATH=/meta/policy
MY_LOGO_PATH=/public/xcvr.svg
MY_OAUTH_CALLBACK=/oauth/callback
MY_JWKS_PATH=/oauth/jwks.json
SESSION_KEY=secret
LRCD_SECRET=secret
```

the first four of course have to do with your postgres instance, i think the
postgres port is just 15432 because i was having conflicts on mac, i don't
think what you set it to is very important. they are also used in the migration
scripts. migratedown and migrateup require you to install golang migrate, and
psql requires you to install psql. MY_NAME, MY_IDENTITY, MY_METADATA_PATH,
MY_TOS_PATH, MY_POLICY_PATH, MY_LOGO_PATH, MY_OAUTH_CALLBACK and MY_JWKS_PATH
are used for oauth, MY_IDENTITY is the hostname, but also the atproto handle of
the backend's repo. the MY_SECRET is variable is thus the backend's app
password. in order to be an oauth client, you need a jwks that you serve at
MY_JWKS_PATH, but of course you need to generate this using haileyok's
atproto-oauth-golang, and then save jwks.json in the top level rvcx directory
as jwks.json. SESSION_KEY and LRCD_SECRET are two more keys that you need to
generate in addition to POSTGRES_PASSWORD. SESSION_KEY encrypts the oauth
session data, and LRCD_SECRET is used to generate nonces that prevent anyone
from submitting other people's unauthenticated messages.

once you have your .env file, you then need to run `sudo docker-compose up -d`
and then `sudo ./migrateup` if it is your first time running the server. if you
need to reset the database, you can do `sudo docker-compose down --volumes`, of
course, be careful with this, rvcx does not currently have a way to backfill
the database!!! these commands are run in the main rcvx directory.

then, you need to `cd server`, and then you can `go run ./cmd` to start the
backend.

i also have included my nginx configuration. after installing nginx, you need
to put that in the conf.d folder, if you're on ubuntu. i think that nginx
differs a bit depending on your distro, you can probably figure it out, i might
be able to help troubleshoot but it'd probably the blind leading the blind
there haha

of course, this is just the backend, so alongside nginx, you should build the
frontend and copy the files to the appropriate location. the frontend uses
sveltekit and i just have it generate a static site bc i don't really care atp,
setup is already contrived enough

as with xcvr (frontend) i think this is under mit but i don't know enough
apropos licensing at this moment in history to fully commit in that direction.

as for contributions, same thing i said in that other readme, i'd love help &
you should let me know in advance so we don't make each other's lives hell :3
