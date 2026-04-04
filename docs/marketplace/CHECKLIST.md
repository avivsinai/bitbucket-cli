# Marketplace Submission Checklist

## Prerequisites (one-time, manual)

- [ ] Create Atlassian developer account at https://developer.atlassian.com
- [ ] Complete KYC/KYB identity verification via Stripe Identity (14-day window)
- [ ] Accept the Marketplace Partner Agreement

## Pre-Submission Validation

- [ ] Open an ECOHELP request to confirm that a non-installable CLI using API
      tokens with scopes (not OAuth 3LO) is eligible for a Cloud listing, or
      request an exemption. See LISTING.md "Authentication Approach" for context.
- [ ] Wait for ECOHELP confirmation before submitting

## Assets (manual)

- [ ] Design logo: 144x144 PNG, transparent background
- [ ] Design banner: 1120x548 PNG, include app name and partner identification
- [ ] Capture 3+ screenshots: 1840x900 PNG
  - Terminal showing `bkt pr list` with color output
  - Terminal showing `bkt pipeline run` and status
  - `bitbucket-pipelines.yml` snippet with bkt in use
- [ ] Prepare cropped screenshots: 580x330 PNG (one per highlight)
- [ ] Optional: record 30-second demo video, upload to YouTube

## Required Documents (all in this directory)

- [x] Privacy policy — PRIVACY.md
- [x] Provider-specific terms (supplements Atlassian's standard end-user
      agreement) — PROVIDER_SPECIFIC_TERMS.md
- [x] Security statement (token handling, data flow, supply chain) —
      SECURITY_STATEMENT.md
- [x] Privacy & Security tab answers — PRIVACY_SECURITY_TAB.md
- [x] Named security contact — avivsinai@gmail.com (in all docs above)

Note: In the partner portal, select Atlassian's **standard customizable
end-user agreement** as the EULA. The provider-specific terms supplement it.
The MIT License remains the source-code license but is not the Marketplace
customer agreement.

## Listing Submission (via portal)

1. [ ] Go to marketplace.atlassian.com → Publish a new app
2. [ ] Select **"My app isn't directly installable"**
3. [ ] Enter app key (immutable — choose carefully, e.g. `com.avivsinai.bkt`)
4. [ ] Enter app name, tagline, summary from LISTING.md
5. [ ] Enter 3 highlights with titles, summaries, and screenshots from LISTING.md
6. [ ] Set compatible applications: **Bitbucket Cloud** only (new DC submissions
       stopped December 16, 2025)
7. [ ] Set pricing: Free
8. [ ] Add documentation URL: https://github.com/avivsinai/bitbucket-cli#readme
9. [ ] Add support URL: https://github.com/avivsinai/bitbucket-cli/issues
10. [ ] Upload privacy policy
11. [ ] Upload visual assets (logo, banner, screenshots)
12. [ ] Complete the app security questionnaire (disclose API token auth model)
13. [ ] Fill in Privacy & Security tab (data handling, token storage, no telemetry)
14. [ ] Submit for review

## Post-Submission

- [ ] Respond to review feedback (auth approach will likely be questioned — see
      LISTING.md "Authentication Approach" section for the prepared response)
- [ ] After approval: add Marketplace badge/link to README.md
- [ ] After approval: update .goreleaser.yaml release footer with Marketplace link

## Timeline Expectations

- ECOHELP response: varies (days to weeks)
- KYC/KYB verification: 2-3 business days
- Listing review: 5-10 business days (can take 2-4 weeks)
- Submissions are processed chronologically; resubmissions restart the queue
