# FRP tunnels

## Nextcloud

Start the local application and its dependencies with:

```sh
docker compose up -d nextcloud nextcloud-cron
```

The local FRP client includes `~/.config/frp/proxies/*.toml`. Install
`nextcloud.toml` in that directory and restart the user service:

```sh
install -m 600 frp/nextcloud.toml ~/.config/frp/proxies/nextcloud.toml
systemctl --user restart frpc
```

The `cd` FRP server must include `{ single = 7008 }` in its `allowPorts` list.
The cloud provider's security group must also permit inbound TCP port `7008`.
With both sides running and that rule in place, Nextcloud is available at:

```text
http://8.137.169.225:7008
```

The FRP control connection uses token authentication and forced TLS. Nextcloud
itself is HTTP on the public TCP port; put it behind the server's HTTPS reverse
proxy before using it for sensitive data over an untrusted network.

## Portainer

Start Portainer and install its proxy definition:

```sh
docker compose up -d portainer
install -m 600 frp/portainer.toml ~/.config/frp/proxies/portainer.toml
systemctl --user restart frpc
```

The `cd` FRP server must include `{ single = 7009 }` in its `allowPorts` list,
and the cloud provider's security group must permit inbound TCP port `7009`.
Portainer is then available at:

```text
https://8.137.169.225:7009
```

Portainer uses a self-signed TLS certificate by default. Because Portainer has
control of the Docker socket, restrict public port `7009` to trusted source IPs
or access it through a VPN rather than exposing it to the whole internet.

## Homepage

Start Homepage and its read-only Docker API proxy, then install the FRP proxy:

```sh
docker compose up -d homepage
install -m 600 frp/homepage.toml ~/.config/frp/proxies/homepage.toml
systemctl --user restart frpc
```

The `cd` FRP server must include `{ single = 7010 }` in its `allowPorts` list,
and the cloud provider's security group must permit inbound TCP port `7010`.
Homepage is then available at:

```text
http://8.137.169.225:7010
```

Homepage does not provide authentication. Restrict public port `7010` to
trusted source IPs or put the dashboard behind an authenticated HTTPS proxy.
