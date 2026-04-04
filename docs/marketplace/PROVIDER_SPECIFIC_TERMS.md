# Provider-Specific Terms for bkt

**Effective date**: 2026-04-04

These Provider-Specific Terms supplement the Atlassian Marketplace standard End
User Agreement for `bkt for Bitbucket`.

## 1. Product form

`bkt` is a standalone command-line interface distributed as downloadable
software. It runs locally on the customer's machine or CI runner. It is not a
hosted SaaS service and does not include a vendor-operated backend.

## 2. Open-source license

The source code for `bkt` is made available under the MIT License at:
https://github.com/avivsinai/bitbucket-cli/blob/master/LICENSE

The MIT License governs rights to the source code. These Provider-Specific
Terms and the Marketplace standard End User Agreement govern the Marketplace
listing, distribution, and any related support or vendor commitments.

## 3. Support

Support is provided on a best-effort basis only.

General support:
https://github.com/avivsinai/bitbucket-cli/issues

Security reports:
https://github.com/avivsinai/bitbucket-cli/blob/master/SECURITY.md

## 4. No hosted processing

`bkt` sends API requests directly from the customer's machine or CI runner to
the customer's selected Bitbucket Cloud endpoints over HTTPS. The provider does
not operate a service that receives, proxies, stores, or analyzes customer
Bitbucket content as part of normal product operation.

## 5. Credentials

Authentication credentials are supplied by the end user and are stored locally
on the end user's device using OS keychain facilities where available, or an
encrypted local file backend when explicitly enabled. Credentials are not
transmitted to provider-operated systems.

## 6. No SLA / uptime commitment

Because `bkt` is downloadable local software and not a hosted service, no
uptime, service availability, backup, disaster recovery, or service-level
commitment is provided.

## 7. Changes

These Provider-Specific Terms may be updated by publishing a revised version
at a stable public URL.
