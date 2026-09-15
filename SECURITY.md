# Security Policy

The Triage team and community take the security and integrity of our software seriously. We appreciate the efforts of security researchers and engineers who discover vulnerabilities and help us protect our users and infrastructure.

---

## Supported Versions

Only the latest minor release of the current major version receives security updates and vulnerability patches.

| Version | Supported   |
| :------ | :---------- |
| v0.2.x  | Supported   |
| < v0.2  | Unsupported |

We strongly encourage all production deployments to upgrade to the latest stable release promptly.

---

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues, discussions, or pull requests.**

### Preferred Method: GitHub Private Vulnerability Reporting

If you have discovered a security vulnerability in Triage, the preferred and most secure reporting method is via GitHub's Private Vulnerability Reporting feature:

1. Navigate to the [Triage Security Advisories](https://github.com/algotyrnt/triage/security/advisories) page.
2. Click **"Report a vulnerability"**.
3. Fill out the advisory form with detailed reproduction steps, impact assessment, and any proof-of-concept (PoC) code or logs.

### Alternative Method: Direct Security Email

If you cannot report via GitHub, you can report vulnerabilities directly to the maintainers at:

```
security@algotyrnt.com
```

Please include:
- A clear description of the vulnerability and affected component (`engine`, `sdk/go`, `dashboard`, `docker image`).
- Detailed steps to reproduce the issue (sample payload, configuration, or environment setup).
- Potential impact and severity assessment (e.g., CVSS estimate).
- Any proposed mitigations or patch suggestions if available.

---

## Response Timeline & SLA

When you submit a vulnerability report:

1. **Initial Acknowledgment:** You will receive an acknowledgment within **48 hours** confirming receipt of your report.
2. **Triaging & Verification:** We will investigate and verify the vulnerability within **5 business days** and communicate our findings.
3. **Patch Development:** We will work on a remediation patch in a private security fork or branch.
4. **Coordinated Disclosure:** Once a fix is verified, a patched release will be published alongside a GitHub Security Advisory crediting the reporter (unless anonymity is requested).

---

## Safe Harbor & Research Guidelines

We consider security research conducted in good faith to be authorized and protected under our safe harbor guidelines. When conducting research:
- Do not access, modify, or exfiltrate data belonging to other users or organizations.
- Do not perform Denial of Service (DoS) attacks against production or demo instances.
- Allow us reasonable time to resolve the issue before disclosing it publicly.
