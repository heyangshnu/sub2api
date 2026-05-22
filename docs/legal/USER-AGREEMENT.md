# Sub2API User Agreement & Privacy Notice

**Version:** 2026-05-22  
**Effective date:** 2026-05-22

> This document is the canonical legal text. The in-app page at `/terms` mirrors this content.

---

## English

### 1. Introduction & Service Description

Sub2API (the **“Service”**) is an API aggregation and relay platform. We provide OpenAI-compatible endpoints so you can access third-party AI models through a unified API key, dashboard, usage metering, and billing.

**We are an intermediary**, not the creator of underlying AI models. Model availability, quality, latency, and pricing may change without notice. The Service is provided on an **“as is”** and **“as available”** basis.

By creating an account, you confirm that you have read, understood, and agree to this User Agreement and Privacy Notice (collectively, the **“Agreement”**).

### 2. Eligibility

You must be at least **16 years old** (or the age of digital consent in your jurisdiction, whichever is higher) and able to form a binding contract. You may not use the Service where prohibited by law.

### 3. Account & Security

- You are responsible for safeguarding your password and API keys.
- You must provide accurate registration information.
- You must notify us promptly of unauthorized access.
- We may suspend or terminate accounts for abuse, fraud, or violation of this Agreement.

### 4. User Data & Privacy

#### 4.1 Data we collect

| Category | Examples | Purpose |
|----------|----------|---------|
| Account | Email, name, password hash | Authentication, support |
| Usage | API requests metadata (model, tokens, timestamps, IP) | Billing, limits, security |
| Payment | Stripe session IDs, amounts (via Stripe) | Top-ups, subscriptions |
| Content | Prompts/completions **transmitted through** the Service to upstream providers | Fulfilling your API requests |

#### 4.2 How we use data

- Operate, secure, and improve the Service  
- Meter usage and enforce balance / subscription caps  
- Comply with law and prevent abuse  
- **We do not sell your personal data.**

#### 4.3 Upstream providers & international transfer

API requests are forwarded to **third-party model providers** (e.g. DeepSeek, OpenAI-compatible hosts). By using the Service, you **authorize** us to transmit your prompts and related metadata to those providers under their terms. Data may be processed in countries other than your own.

#### 4.4 Retention

We retain account and billing records as needed for operations, tax, and dispute resolution. Request logs may be stored in our database (SQLite) and cache (Redis) according to our retention practices.

#### 4.5 Your rights (GDPR / similar)

Where applicable, you may request **access, correction, deletion, restriction, or portability** of personal data, and object to certain processing. Contact the operator email shown on the Service website. You may lodge a complaint with your local supervisory authority.

### 5. Payment & Financial Risk Disclosure

- Balances and prices are shown in **USD** unless stated otherwise.
- **Prepaid balance** is consumed per API usage; unused balance may not be refundable except where required by law or explicitly stated.
- **Stripe** processes card payments; we do not store full card numbers.
- Payment failures, chargebacks, or webhook delays may delay crediting your account.
- **Subscription tiers** (if enabled) limit models and periodic spend caps; they do not replace metered balance deductions unless stated.
- **No investment or financial advice** is provided. You accept pricing volatility and upstream cost changes.
- Taxes, currency conversion, and bank fees are your responsibility.

### 6. Acceptable Use

You agree **not** to:

- Violate laws or third-party rights  
- Send malware, spam, or illegal content  
- Attempt to bypass rate limits, billing, or security  
- Resell or share API keys in violation of fair use  
- Use the Service for high-risk activities (medical, legal, or safety-critical decisions without human review)

### 7. Intellectual Property

The Service software, branding, and documentation are owned by the operator. You retain rights to your own input content, subject to licenses you grant to us and upstream providers to operate the Service.

### 8. Disclaimer of Warranties

TO THE MAXIMUM EXTENT PERMITTED BY LAW, THE SERVICE IS PROVIDED **WITHOUT WARRANTIES** OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, AND NON-INFRINGEMENT. WE DO NOT GUARANTEE UNINTERRUPTED, ERROR-FREE, OR SECURE OPERATION.

### 9. Limitation of Liability

TO THE MAXIMUM EXTENT PERMITTED BY LAW, WE SHALL NOT BE LIABLE FOR INDIRECT, INCIDENTAL, SPECIAL, CONSEQUENTIAL, OR PUNITIVE DAMAGES, OR LOSS OF PROFITS, DATA, OR GOODWILL. OUR AGGREGATE LIABILITY FOR ANY CLAIM ARISING FROM THE SERVICE SHALL NOT EXCEED THE GREATER OF **(A) USD $50** OR **(B) THE AMOUNTS YOU PAID US IN THE 12 MONTHS** BEFORE THE CLAIM.

Some jurisdictions do not allow certain limitations; in those cases, our liability is limited to the fullest extent permitted.

### 10. Indemnification

You agree to indemnify and hold harmless the operator from claims arising from your use of the Service, your content, or your violation of this Agreement.

### 11. Changes

We may update this Agreement. Material changes will be reflected by a new **version** date. Continued use after the effective date constitutes acceptance. Registration requires accepting the current version.

### 12. Governing Law & Disputes

Unless mandatory local law requires otherwise, this Agreement is governed by the laws of the **operator’s principal place of business**, without regard to conflict-of-law rules. Disputes shall be resolved in the courts of that jurisdiction, or through binding arbitration if we publish such a process.

### 13. Contact

For privacy or legal inquiries, use the contact email published on **cloudtoken.uk** or your deployment’s support channel.

---

**Version ID for registration:** `2026-05-22`
