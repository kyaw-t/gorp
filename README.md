# Go Registry Proxy

## Warning: This is an initial prototype of a registry proxy

## Purpose

The registry proxy acts as intermediary between your package manager and the registries you're trying to download from.

Gorp enables you to specify fallback registries, as well as fine-scoping all your packages.

## Suppoorted Package Managers

As of now only Node package managers are supported, but there are plans to add support for other package mangers (pip).

### Node

The following package managers are supported. Gorp will work with yarn version older than 2, however the generated lockfile (yarn.lock) will not accurately reflect the resolved source of the package. This will not only produce challenges when sharing the lock file between various developers in a team but ultimately defeat the purpose of a lockfile. Therefore, it is recommended to not use Gorp with yarn versions older than 2.

- npm
- yarn (^2.0.0)
- pnpm
