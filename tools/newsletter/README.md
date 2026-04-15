# Newsletter Subscription System

Newsletter for Aetheris/CoRag - the durable execution runtime for AI agents.

## Overview

This directory contains the infrastructure for the Aetheris newsletter subscription system.

## Directory Structure

```
tools/newsletter/
├── README.md              # This file
├── subscribe.html         # Email collection landing page
├── templates/             # Newsletter content templates
│   ├── default.md         # Default newsletter template
│   ├── release-announce.md # Release announcement template
│   └── dev-update.md      # Development update template
├── integrations/          # Email service integration guides
│   ├── mailchimp.md       # Mailchimp integration guide
│   ├── buttondown.md      # Buttondown integration guide
│   └── resend.md          # Resend integration guide
└── LIST.md                # Newsletter subscriber list management
```

## Quick Start

### 1. Email Collection

Deploy `subscribe.html` as a landing page to collect email addresses. It can be
integrated with any email service via their embed forms or API.

### 2. Choose an Email Service

See the integrations folder for setup guides:
- **Mailchimp**: Best for beginners, has free tier
- **Buttondown**: Simple, developer-friendly, cheap
- **Resend**: Modern API, great for developers

### 3. Send Newsletters

1. Copy a template from `templates/`
2. Customize the content
3. Send via your email service or export as HTML

## Newsletter Cadence

| Type | Frequency | Content |
|------|-----------|---------|
| Release Announcements | As needed | New features, major updates |
| Development Updates | Bi-weekly | Progress, code insights |
| Monthly Digest | Monthly | Top discussions, community highlights |

## Content Guidelines

- Keep subject lines under 50 characters
- Use actionable CTAs (subscribe to GitHub, try the quickstart)
- Include code examples when relevant
- Highlight community contributions
- Add images sparingly (many email clients block images by default)

## Best Practices

1. **Subject Lines**: Clear, benefit-driven, create urgency
   - ✅ "Aetheris 2.0: Crash Recovery That Actually Works"
   - ❌ "Monthly Update"

2. **Personalization**: Address subscribers by name when possible

3. **Mobile Optimization**: Keep content under 600px wide

4. **Unsubscribe Link**: Always include (required for compliance)

## Example Subject Lines

- "🚀 Aetheris 1.5: At-Most-Once Tool Execution is Here"
- "How We Built Deterministic Replay for AI Agents"
- "Community Spotlight: Building Compliance Pipelines with Aetheris"
- "5 Patterns for Durable Agent Execution"
- "Behind the Scenes: Aetheris Architecture Deep Dive"

## API Integration

For programmatic newsletter management:

```bash
# Example: Add subscriber via Buttondown API
curl -X POST "https://api.buttondown.email/v1/subscribers" \
  -H "Authorization: Token YOUR_API_TOKEN" \
  -d "{\"email\": \"user@example.com\", \"tags\": [\"aetheris-users\"]}"
```

## Getting Help

- GitHub Issues: https://github.com/Colin4k1024/Aetheris/issues
- Discord: https://discord.gg/PrrK2Mua
- Documentation: https://docs.aetheris.ai
