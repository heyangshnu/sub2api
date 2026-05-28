"use client";

import { TermsContentZh } from "@/components/legal/terms-content-zh";
import { useLocale } from "@/lib/i18n";
import { TERMS_LAST_UPDATED, TERMS_VERSION } from "@/lib/terms";

export function TermsContent() {
  const { locale } = useLocale();
  if (locale === "zh") {
    return <TermsContentZh />;
  }

  return (
    <article className="prose prose-slate max-w-none text-sm prose-headings:text-slate-900 prose-p:text-slate-700">
      <p className="text-xs text-slate-500">
        Version {TERMS_VERSION} · Last updated {TERMS_LAST_UPDATED}
      </p>
      {/* English body — zh uses TermsContentZh */}

      <h2>1. Service Description</h2>
      <p>
        Sub2API (the &quot;Service&quot;) is an API aggregation and relay platform providing
        OpenAI-compatible endpoints, API keys, usage metering, and billing. We are an{" "}
        <strong>intermediary</strong>, not the creator of underlying AI models. Model availability,
        quality, latency, and pricing may change without notice.
      </p>

      <h2>2. Acceptance</h2>
      <p>
        By registering, you confirm that you have read, understood, and agree to this User
        Agreement and Privacy Notice. You must accept version <code>{TERMS_VERSION}</code> to create
        an account.
      </p>

      <h2>3. Eligibility</h2>
      <p>
        You must be at least 16 years old (or the age of digital consent in your jurisdiction) and
        legally able to enter a contract. You may not use the Service where prohibited by law.
      </p>

      <h2>4. User Data &amp; Privacy</h2>
      <h3>4.1 Data we collect</h3>
      <ul>
        <li>
          <strong>Account data:</strong> email, display name, password hash — for authentication
          and support.
        </li>
        <li>
          <strong>Usage data:</strong> API metadata (model, tokens, timestamps, IP, request IDs) —
          for billing, limits, and security.
        </li>
        <li>
          <strong>Payment data:</strong> processed by Stripe; we do not store full card numbers.
        </li>
        <li>
          <strong>Content:</strong> prompts and completions transmitted through the Service to
          upstream AI providers to fulfill your requests.
        </li>
      </ul>
      <h3>4.2 How we use data</h3>
      <p>
        To operate, secure, and improve the Service; meter usage; enforce balance and subscription
        limits; comply with law; prevent abuse. <strong>We do not sell your personal data.</strong>
      </p>
      <h3>4.3 Authorization to process &amp; transfer</h3>
      <p>
        You <strong>authorize</strong> us to collect, store, process, and forward your API requests
        (including prompt content and metadata) to third-party model providers. Data may be processed
        in countries other than your own. Each provider applies its own terms and privacy policy.
      </p>
      <h3>4.4 Your rights</h3>
      <p>
        Where applicable (e.g. GDPR), you may request access, correction, deletion, restriction, or
        portability of personal data. Contact the operator via the email published on the Service
        website. You may lodge a complaint with your local data protection authority.
      </p>

      <h2>5. Payment &amp; Financial Risk</h2>
      <ul>
        <li>Balances and prices are in <strong>USD</strong> unless stated otherwise.</li>
        <li>
          Prepaid balance is consumed per API usage. Unused balance is generally{" "}
          <strong>non-refundable</strong> except where required by law.
        </li>
        <li>
          Stripe processes payments. Delays, chargebacks, or webhook failures may delay account
          credits.
        </li>
        <li>
          Subscriptions (if enabled) limit allowed models and periodic spend caps; metered balance
          deductions may still apply.
        </li>
        <li>No investment or financial advice is provided. You accept pricing and upstream cost changes.</li>
      </ul>

      <h2>6. Acceptable Use</h2>
      <p>You must not violate laws, abuse the API, bypass billing or security, or use the Service for unlawful or high-risk automated decisions without human review.</p>

      <h2>7. Disclaimers &amp; Liability</h2>
      <p>
        THE SERVICE IS PROVIDED &quot;AS IS&quot; WITHOUT WARRANTIES. TO THE MAXIMUM EXTENT
        PERMITTED BY LAW, OUR AGGREGATE LIABILITY SHALL NOT EXCEED THE GREATER OF USD $50 OR THE
        AMOUNTS YOU PAID IN THE 12 MONTHS BEFORE A CLAIM. Some jurisdictions do not allow certain
        limitations.
      </p>

      <h2>8. Changes</h2>
      <p>
        We may update this Agreement. A new version ID will be required for new registrations.
        Continued use after the effective date constitutes acceptance.
      </p>
    </article>
  );
}
