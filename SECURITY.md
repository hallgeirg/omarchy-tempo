# Security

Never include API tokens, credentials files, private time entries, or unredacted exports in public issues.

For a suspected vulnerability, use GitHub's private vulnerability reporting if available. Otherwise contact the maintainer privately before publishing exploitable details. This is an early personal project; no response-time guarantee is offered.

Tokens are stored locally as plaintext with mode 0600. The cache contains account metadata and time entries and also uses private file permissions. Filesystem permissions do not protect against other software running as your user. Requests use HTTPS; the application does not send telemetry.

Omarchy plugins execute code on your machine. Review source and binaries before enabling plugins. The bundled executable can be rebuilt using the documented Go toolchain.
