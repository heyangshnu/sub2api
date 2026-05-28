import { TERMS_LAST_UPDATED, TERMS_VERSION } from "@/lib/terms";

export function TermsContentZh() {
  return (
    <article className="prose prose-slate max-w-none text-sm prose-headings:text-slate-900 prose-p:text-slate-700">
      <p className="text-xs text-slate-500">
        版本 {TERMS_VERSION} · 更新日期 {TERMS_LAST_UPDATED}
      </p>

      <h2>1. 服务说明</h2>
      <p>
        Sub2API（「本服务」）是 API 聚合与转发平台，提供 OpenAI 兼容接口、API 密钥、用量计量与计费。我们是
        <strong>中间方</strong>，并非底层 AI 模型的提供方。模型可用性、质量、延迟与价格可能随时变更。
      </p>

      <h2>2. 接受条款</h2>
      <p>
        注册即表示您已阅读、理解并同意本用户协议与隐私说明。创建账户须接受版本 <code>{TERMS_VERSION}</code>。
      </p>

      <h2>3. 资格</h2>
      <p>
        您须年满 16 周岁（或您所在司法辖区规定的数字同意年龄）并具备缔约能力。在法律禁止的地区不得使用本服务。
      </p>

      <h2>4. 用户数据与隐私</h2>
      <h3>4.1 我们收集的数据</h3>
      <ul>
        <li>
          <strong>账户数据：</strong>邮箱、显示名、密码哈希 — 用于认证与支持。
        </li>
        <li>
          <strong>用量数据：</strong>API 元数据（模型、Token、时间戳、IP、请求 ID）— 用于计费、限额与安全。
        </li>
        <li>
          <strong>支付数据：</strong>由 Stripe 处理；我们不存储完整卡号。
        </li>
        <li>
          <strong>内容：</strong>通过本服务传输至上游 AI 提供商的提示与回复，用于完成您的请求。
        </li>
      </ul>
      <h3>4.2 数据用途</h3>
      <p>
        用于运营、安全与改进服务；计量用量；执行余额与订阅限制；遵守法律；防止滥用。
        <strong>我们不出售您的个人数据。</strong>
      </p>
      <h3>4.3 处理与跨境传输授权</h3>
      <p>
        您<strong>授权</strong>我们收集、存储、处理并将您的 API 请求（含提示内容与元数据）转发至第三方模型提供商。数据可能在您所在国以外处理。各提供商适用其自有条款与隐私政策。
      </p>
      <h3>4.4 您的权利</h3>
      <p>
        在适用法律下（如 GDPR），您可请求访问、更正、删除、限制或转移个人数据。请通过本服务网站公布的邮箱联系运营方。您可向当地数据保护机构投诉。
      </p>

      <h2>5. 支付与资金风险</h2>
      <ul>
        <li>余额与价格以 <strong>美元 (USD)</strong> 计，除非另有说明。</li>
        <li>预付费余额按 API 用量扣减。未使用余额通常<strong>不可退款</strong>，法律另有规定除外。</li>
        <li>Stripe 处理支付。延迟、拒付或 Webhook 失败可能导致到账延迟。</li>
        <li>订阅（若启用）限制可用模型与周期消费上限；仍可能按量从余额扣费。</li>
        <li>本服务不提供投资或财务建议。您接受定价与上游成本变动。</li>
      </ul>

      <h2>6. 可接受使用</h2>
      <p>
        不得违法使用、滥用 API、绕过计费或安全，或在无人工复核的情况下将本服务用于违法或高风险自动决策。
      </p>

      <h2>7. 免责声明与责任限制</h2>
      <p>
        本服务按「现状」提供，不作任何保证。在法律允许的最大范围内，我们累计责任不超过 50 美元或您提出索赔前 12 个月内支付金额的较高者。部分司法辖区不允许特定限制。
      </p>

      <h2>8. 变更</h2>
      <p>我们可能更新本协议。新版本 ID 将用于新注册用户。生效日后继续使用视为接受。</p>
    </article>
  );
}
