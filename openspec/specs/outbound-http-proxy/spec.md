# Outbound HTTP Proxy Specification

## Purpose

Lets Databasus run in networks where outbound traffic must leave through a proxy. Every storage and notifier that talks HTTP reaches its endpoint through the proxy the operator configured in the process environment, and goes direct for the hosts the operator excluded.

## Requirements

### Requirement: A proxied HTTPS connection keeps TLS end to end

When an S3 or Azure Blob storage or a Mattermost notifier sends a request to an `https` endpoint through a proxy, it SHALL ask the proxy only for a tunnel to the endpoint's host and port, and SHALL establish the TLS session with the endpoint inside that tunnel. The proxy SHALL NOT receive the request line, the headers, the body or the credentials sent to the endpoint. The endpoint's certificate SHALL be verified exactly as it is without a proxy: with TLS verification on, a certificate that is not trusted for the endpoint fails the request; with TLS verification turned off in the storage or notifier settings, the request succeeds as it does without a proxy.

#### Scenario: Credentials stay inside the tunnel

- **WHEN** a storage or notifier with an `https` endpoint sends a request with credentials through a proxy
- **THEN** the proxy receives a single tunnel request naming the endpoint's host and port, and the request with its credentials reaches only the endpoint

#### Scenario: Untrusted endpoint certificate through a proxy

- **WHEN** a storage or notifier with TLS verification on connects through a proxy to an endpoint whose certificate is not trusted
- **THEN** the request fails with a certificate error, as it would without a proxy

#### Scenario: TLS verification turned off through a proxy

- **WHEN** an S3 storage or a Mattermost notifier with TLS verification turned off connects through a proxy to an endpoint whose certificate is not trusted
- **THEN** the request reaches the endpoint, as it would without a proxy

### Requirement: Supported HTTP storages and notifiers use the proxy from the environment

S3 and Azure Blob storages, and every notifier that talks HTTP, SHALL send their requests through the proxy named by the standard proxy variables of the process environment: `HTTPS_PROXY` for `https` endpoints and `HTTP_PROXY` for `http` endpoints, in upper or lower case. They SHALL connect directly to a host that matches `NO_PROXY`, and to localhost and loopback addresses. When no proxy variable is set, they SHALL connect directly. This SHALL hold for every request these storages make: uploads, downloads, existence checks, deletions and the connection test. It SHALL hold for a Mattermost notifier whether TLS verification is on or off. A notifier configured with its own proxy, such as a Telegram notifier with a proxy enabled, SHALL use that proxy instead of the environment's.

#### Scenario: S3 upload behind a proxy

- **WHEN** `HTTPS_PROXY` names a proxy, an S3 storage points at an `https` endpoint outside `NO_PROXY`, and a backup is uploaded to it
- **THEN** the upload goes through the proxy, and no direct connection to the endpoint is attempted

#### Scenario: Storage endpoint excluded from the proxy

- **WHEN** `HTTPS_PROXY` names a proxy and `NO_PROXY` lists the host of an S3 storage endpoint
- **THEN** requests to that storage connect to the endpoint directly

#### Scenario: No proxy configured

- **WHEN** no proxy variable is set in the process environment
- **THEN** S3 and Azure Blob storages and all HTTP notifiers connect to their endpoints directly, as before this change

#### Scenario: Azure Blob behind a proxy

- **WHEN** `HTTPS_PROXY` names a proxy and a backup is uploaded to an Azure Blob storage
- **THEN** the upload goes through the proxy

#### Scenario: Mattermost notifier with TLS verification turned off

- **WHEN** `HTTPS_PROXY` names a proxy and a Mattermost notifier with TLS verification turned off sends a notification
- **THEN** the notification goes through the proxy

#### Scenario: Notifier with its own proxy

- **WHEN** `HTTPS_PROXY` names one proxy and a Telegram notifier has a different proxy enabled in its settings
- **THEN** the notification goes through the proxy from the notifier's settings

#### Scenario: Storage connection test behind a proxy

- **WHEN** `HTTPS_PROXY` names a proxy and the user tests the connection of an S3 storage
- **THEN** the test request goes through the proxy, so the result matches what scheduled backups will experience
