# Security Policy

## Supported versions

Security fixes go into the latest minor release. Older minors are not
patched, so upgrade to the latest tag before reporting a problem.

| Version | Supported |
| ------- | --------- |
| 1.4.x   | yes       |
| < 1.4   | no        |

## Reporting a vulnerability

Report security vulnerabilities via [GitHub private vulnerability
reporting](https://github.com/italia/developers-italia-api/security/advisories/new).
The report stays private between you and the maintainers until a fix is
released.

If you would rather use email, write to <fb@fabiobonelli.it> encrypted
with this OpenPGP key:

```text
AA6C A187 99DC 9000 F291  2B0C 909A 463B A33D 0B45
```

The key is on <https://keys.openpgp.org> and at
<https://github.com/bfabio.gpg>.

Do not open a public issue, and do not disclose the problem elsewhere
before a fix is available.

Useful things to include, as far as you can:

- the affected version or commit
- the endpoint and the request that triggers the problem
- what an attacker gets out of it
- a way to reproduce it

You can expect an acknowledgement within 14 days. If the report is
accepted, we will agree a disclosure date with you and credit you in the
advisory unless you prefer otherwise.

## Scope

This policy covers the API server code in this repository and the container
images and Helm chart published from it.
