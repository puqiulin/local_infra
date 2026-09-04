# New API

New API runs on Beelink using the official `calciumion/new-api:latest`
image. Its SQLite database and logs are stored in named Docker volumes.

The web interface is intentionally bound to loopback. Open an SSH tunnel from
the client machine:

```sh
ssh -N -L 3000:127.0.0.1:3000 beelink
```

Keep that command running, then open:

```text
http://localhost:3000
```

Start or update the service on Beelink with:

```sh
docker compose pull new-api
docker compose up -d new-api
```

Check its state and recent logs with:

```sh
docker compose ps new-api
docker compose logs --tail=100 new-api
```

`NEW_API_SESSION_SECRET` and `NEW_API_CRYPTO_SECRET` must be set in the ignored
`.env` file before starting the service. Never commit those values or the admin
password.
