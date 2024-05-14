# shareasecret

> [!NOTE]
> shareasecret is still under active development and shouldn't yet be considered to be in an alpha state, let alone anything more.

Client-side encrypted, time limited, opening count restricted shareable links. Securely share secrets with others
without the plaintext version of your secret ever leaving your device.

Due to its dependency on client side encryption libraries a browser supporting JavaScript is required in order to use
shareasecret once deployed.

![A screenshot of the landing page of a hosted shareasecret instance](/assets/screenshot.png)

## Disclaimer

As per the license agreement you agree to when you utilise this software, no warranty is made as to its effectiveness
or security. If your risk assessment requires you to use proven software, find alternatives that have passed in-depth
security audits from reputable security firms.

For more information, see the [LICENSE](/LICENSE).

## Issues?

Please report any issues (with reproduction steps if applicable) in this repository's issue tracker.

## How does it work?

shareasecret is a Go application built with the Templ templating language and enhanced with vanilla JavaScript and CSS.
Its dependencies are deliberately kept slim for a faster and more secure user experience. In addition, a strict
[Content Security Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP) is in place to prevent loading of
unknown and potentially malicious resources.

Secrets are encrypted on the client side using the [Web Crypto API](https://developer.mozilla.org/en-US/docs/Web/API/Web_Crypto_API)
available in most modern browsers.

When you create a secret with your provided "encryption password", a 256 bit AES-GCM encryption key is derived from it
using the PBKDF2 key derivation function. This key is then used in the AES-GCM encryption algorithm to encrypt the plain
text secret resulting in a text blob consisting of three parts: the plaintext 128 bit cryptographically random generated
salt for the PBKDF2 function; the plaintext 96 bit cryptographically random generated IV (initialization vector) for
the AES-GCM encryption algorithm; and the encrypted cipher text of the original secret.

This text blob is what is then sent to the server to be persisted as a "secret". At this point, two 192 bit
cryptographically random identifiers are created: one for viewing and one for management. The viewing id is
what is shared and used to access the decryption form, whilst the management id is used to perform management functions
such as deleting the secret and to view analytics such as how many times the secret has been opened.

When someone with the "viewing id" link accesses the page they are prompted to enter the original encryption password.
Then, the same cycle as before begins, except the derived key is used to decrypt the cipher text to plaintext instead
of encrypting it from plaintext to cipher text.

## Installation

shareasecret is a Go application. As such, it is distributed as a single binary. A simple Docker wrapper around the
binary also exists for those who already utilise such a hosting method.

### Pre-requisites

There is one hard pre-requisite in order to run shareasecret. You **must** have a reverse proxy capable of serving
sites solely in a HTTPS context. It can utilise self signed certificates or certificates from services such as
LetsEncrypt but is an absolute requirement for security purposes; the protection of your users; and, most importantly,
for the WebCrypto engine which powers the entire application to _actually_ work.

### Source

Building the application from source requires Go 1.22.

When running for development purposes on a localhost IP address some browsers will consider it secure and provide APIs
(such as Web Crypto) that it otherwise wouldn't without being on a secure site.

1. Fetch the repository: `git clone https://github.com/lsymds/shareasecret.git && cd shareasecret/`
2. Install the required Go dependencies: `go mod download`
3. Copy the `.env.example` file to `.env` and update it: `cp .env.example .env`
4. Build the application: `go build -o shareasecret .`
5. Execute the application: `./shareasecret`

To compile the templates if you wish to make any front-end changes:

1. Install the Templ CLI: `go install github.com/a-h/templ/cmd/templ@latest`
2. Compile the templates: `templ generate`

### Binary

Requirements: `bzip2`

1. Download the latest release for your operating system and architecture from the releases page:
   [https://github.com/lsymds/shareasecret/releases](https://github.com/lsymds/shareasecret/releases).
2. Extract the archive: `bunzip2 shareasecret-linux-amd64-v1.0.0.bzip2`
3. Set the environment variables according to the configuration section below.
4. Execute the extracted binary: `./shareasecret-linux-amd64-v1.0.0`
5. Profit.

### Docker

Requirements: Docker or Podman

```
docker run --rm -e SHAREASECRET_BASE_URL=http://127.0.0.1:8994 -p 8994:8994 ghcr.io/lsymds/shareasecret:latest
```

By default, the SQLite database that powers a shareasecret instance is persisted under the `/data` directory. A volume
mount is created at the same location also. Feel free to override this with your own volume. The database files will
be created automatically.

The Docker image runs under a user and group id of 1000 by default. These can be overridden by setting the UID/GUID
environment variables when running the container. If you are running rootless Podman and want them to map to your
rootless user's ids, set them to 0:0 (root:root).

## Configuration

### Environment Variables

shareasecret configuration is achieved entirely via environment variables.

If you are running via the executable and _really_ need to configure it via a file, place a `.env` file in your working
directory.

- `SHAREASECRET_DB_PATH` - the path to the database file. Will be created if it doesn't exist. Accompanying `shm` and
  `wal` files will be created alongside it.
- `SHAREASECRET_BASE_URL` - the base URL that shareasecret will be running under i.e. `https://secret.mycompany.example`
- `SHAREASECRET_LISTENING_ADDR` - the address (including port) that the server will listen on. Defaults to
  `127.0.0.1:8994`.
- `SHAREASECRET_SECRET_CREATION_IP_RESTRICTIONS` - a string containing a comma separated list of IP addresses (v4 or v6)
  that are permitted to create secrets. Leaving this empty or not specifying it (the default) will result in an instance
  where anyone can create secrets. Requesting IP addresses are sourced from the `X-Forwarded-For` header. If you need to
  customise this (i.e. to use Cloudflare's `CF-Connecting-IP` instead), use a reverse proxy such as Caddy or Nginx.
