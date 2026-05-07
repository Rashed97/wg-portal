# WireGuard Portal v2 — bnet/awg fork

> **Fork notice — Bitfissure BNet:**
> This branch (`bnet/awg`) adds an **[AmneziaWG](https://github.com/amnezia-vpn/amneziawg-go)** backend
> alongside the existing wgctrl/MikroTik/pfSense backends. AmneziaWG is a WireGuard
> protocol fork that adds traffic obfuscation (V2 params: Jc/Jmin/Jmax/S1-S4/H1-H4/I1-I5)
> to evade DPI-based WireGuard fingerprinting, useful in restrictive networks.
>
> Architecture: a new `AmneziaController` (`internal/adapters/wgcontroller/amnezia.go`)
> shells out to `awg` and `awg-quick` rather than calling netlink directly — the
> wgctrl-go library doesn't recognize AWG's extended UAPI keys. This is ~3× cheaper
> to maintain than a netlink port. The Docker image bundles upstream amneziawg-tools
> compiled from a pinned tag.
>
> Operator-facing changes:
> 1. New `Backend = amneziawg` option for interfaces.
> 2. 16 V2 obfuscation params editable in the InterfaceEditModal.
> 3. Peer config download (`?style=amneziawg`) embeds those params in the
>    `[Interface]` section. Vanilla `wg-quick` style auto-promotes to `amneziawg`
>    when the server interface is AWG-backed (vanilla wg clients ignore the extra
>    fields harmlessly).
>
> All vanilla WireGuard flows remain unchanged. Track upstream merge status in
> bead `BNet-a2rn` (epic).

[![Build Status](https://github.com/h44z/wg-portal/actions/workflows/docker-publish.yml/badge.svg?event=push)](https://github.com/h44z/wg-portal/actions/workflows/docker-publish.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](https://opensource.org/licenses/MIT)
![GitHub last commit](https://img.shields.io/github/last-commit/h44z/wg-portal/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/h44z/wg-portal)](https://goreportcard.com/report/github.com/h44z/wg-portal)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/h44z/wg-portal)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/h44z/wg-portal)
[![Docker Pulls](https://img.shields.io/docker/pulls/h44z/wg-portal.svg)](https://hub.docker.com/r/wgportal/wg-portal/)

## Introduction
<!-- Text from this line # is included in docs/documentation/overview.md -->
**WireGuard Portal** is a simple, web-based configuration portal for [WireGuard](https://wireguard.com) server management.
The portal uses the WireGuard [wgctrl](https://github.com/WireGuard/wgctrl-go) library to manage existing VPN
interfaces. This allows for the seamless activation or deactivation of new users without disturbing existing VPN
connections.

The configuration portal supports using a database (SQLite, MySQL, MsSQL, or Postgres), OAuth or LDAP
(Active Directory or OpenLDAP) as a user source for authentication and profile data.

## Features

* Self-hosted - the whole application is a single binary
* Responsive multi-language web UI with dark-mode written in Vue.js
* Automatically selects IP from the network pool assigned to the client
* QR-Code for convenient mobile client configuration
* Sends email to the client with QR-code and client config
* Enable / Disable clients seamlessly
* Generation of wg-quick configuration file (`wgX.conf`) if required
* User authentication (database, OAuth, or LDAP), Passkey support
* IPv6 ready
* Docker ready
* Can be used with existing WireGuard setups
* Support for multiple WireGuard interfaces
* Supports multiple WireGuard backends (wgctrl, MikroTik, or pfSense)
* Peer Expiry Feature
* Handles route and DNS settings like wg-quick does
* Exposes Prometheus metrics for monitoring and alerting
* REST API for management and client deployment
* Webhook for custom actions on peer, interface, or user updates

<!-- Text to this line # is included in docs/documentation/overview.md -->
![Screenshot](docs/assets/images/screenshot.png)

## Documentation

For the complete documentation visit [wgportal.org](https://wgportal.org).

## What is out of scope

* Automatic generation or application of any `iptables` or `nftables` rules.
* Support for operating systems other than linux.
* Automatic import of private keys of an existing WireGuard setup.

## Application stack

* [wgctrl-go](https://github.com/WireGuard/wgctrl-go) and [netlink](https://github.com/vishvananda/netlink) for interface handling
* [Bootstrap](https://getbootstrap.com/), for the HTML templates
* [Vue.js](https://vuejs.org/), for the frontend

## License

* MIT License. [MIT](LICENSE.txt) or <https://opensource.org/licenses/MIT>

## Contributors and Sponsors

Thanks so much for all your contributions! They’re truly appreciated and help keep WireGuard Portal moving ahead.

<a href="https://github.com/h44z/wg-portal/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=h44z/wg-portal" />
</a>

Want to support the project? You can buy me a coffee or join as a contributor - every bit of support helps! 
[Become a sponsor!](https://github.com/sponsors/h44z)


> [!IMPORTANT]
> Since the project was accepted by the Docker-Sponsored Open Source Program, the Docker image location has moved to [wgportal/wg-portal](https://hub.docker.com/r/wgportal/wg-portal).
> Please update the Docker image from **h44z/wg-portal** to **wgportal/wg-portal**.
