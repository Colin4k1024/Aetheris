# Resend Integration Guide

Resend is a modern email API built for developers, perfect for sending newsletters programmatically.

## Prerequisites

- Resend account (free tier: 3,000 emails/day)
- API key from Resend dashboard
- Verified domain (required for sending)

## Option 1: Simple Email Collection

For collecting emails, combine Resend with a form handler:

### 1. Create a Landing Page Handler

```go
package handler

import (
    "github.com/cloudwego/hertz/pkg/app"
)

type SubscribeRequest struct {
    Email string `json:"email" binding:"required,email"`
}

// HandleSubscribe processes email subscriptions
func HandleSubscribe(ctx context.Context, c *app.RequestContext) {
    var req SubscribeRequest
    if err := c.BindJSON(&req); err != nil {
        c.JSON(400, map[string]string{"error": "invalid email"})
        return
    }
    
    // Store subscriber in database or forward to your email service
    // (Resend is for sending, not collecting subscriptions)
    
    c.JSON(200, map[string]string{"message": "subscribed"})
}
```

## Option 2: Send Newsletter with Resend

### 1. Install Resend SDK

```bash
go get github.com/resend/resend-go
```

### 2. Send Basic Newsletter

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/resend/resend-go"
)

func sendNewsletter() error {
    client := resend.NewClient("re_YOUR_API_KEY")
    
    ctx := context.Background()
    
    params := &resend.SendEmailRequest{
        From:    "Aetheris Team <newsletter@aetheris.ai>",
        To:      []string{"subscriber@example.com"},
        Subject: "What's New in Aetheris",
        Html:    `<h1>Aetheris Update</h1><p>Content here...</p>`,
    }
    
    resp, err := client.Email.Send(ctx, params)
    if err != nil {
        return fmt.Errorf("failed to send email: %w", err)
    }
    
    fmt.Printf("Email sent: %s\n", resp.Id)
    return nil
}
```

### 3. Send to Multiple Recipients (Batched)

```go
func sendBulkNewsletter(subscribers []string, subject, htmlBody string) error {
    client := resend.NewClient("re_YOUR_API_KEY")
    ctx := context.Background()
    
    // Resend supports up to 100 recipients per batch
    batchSize := 100
    
    for i := 0; i < len(subscribers); i += batchSize {
        end := i + batchSize
        if end > len(subscribers) {
            end = len(subscribers)
        }
        
        params := &resend.SendEmailRequest{
            From:    "Aetheris Team <newsletter@aetheris.ai>",
            To:      subscribers[i:end],
            Subject: subject,
            Html:    htmlBody,
        }
        
        _, err := client.Email.Send(ctx, params)
        if err != nil {
            log.Printf("Failed to send batch %d-%d: %v", i, end, err)
            continue
        }
        
        fmt.Printf("Sent batch %d-%d\n", i, end)
    }
    
    return nil
}
```

### 4. Send with Unsubscribe Header (CAN-SPAM)

```go
func sendNewsletterWithUnsubscribe() error {
    client := resend.NewClient("re_YOUR_API_KEY")
    ctx := context.Background()
    
    // Use List-Unsubscribe header for CAN-SPAM compliance
    params := &resend.SendEmailRequest{
        From:    "Aetheris Team <newsletter@aetheris.ai>",
        To:      []string{"subscriber@example.com"},
        Subject: "What's New in Aetheris",
        Html: `<h1>Aetheris Update</h1>
<p>Content here...</p>
<p><a href="mailto:unsubscribe@aetheris.ai?subject=unsubscribe">Unsubscribe</a></p>`,
        Headers: map[string]string{
            "List-Unsubscribe": "mailto:unsubscribe@aetheris.ai",
            "List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
        },
    }
    
    _, err := client.Email.Send(ctx, params)
    return err
}
```

## Option 3: React Email Templates

Resend provides React Email for beautiful email templates.

### 1. Install React Email

```bash
npm install react react-dom @react-email/components
```

### 2. Create Email Template

```jsx
// emails/Newsletter.tsx
import { Html, Head, Body, Container, Section, Text, Button, Link } from '@react-email/components';

export default function Newsletter({ title, content, unsubscribeUrl }) {
    return (
        <Html>
            <Head />
            <Body style={{ backgroundColor: '#f5f5f5' }}>
                <Container style={{ backgroundColor: '#fff', padding: '40px', borderRadius: '8px' }}>
                    <Section>
                        <Text style={{ fontSize: '24px', fontWeight: 'bold' }}>⚡ Aetheris</Text>
                        <Text style={{ fontSize: '18px', fontWeight: 'bold', marginTop: '20px' }}>{title}</Text>
                        <Text style={{ color: '#666', lineHeight: '1.6' }}>{content}</Text>
                        <Button href="https://docs.aetheris.ai" style={{ backgroundColor: '#3b82f6', color: '#fff', padding: '12px 24px', borderRadius: '6px' }}>
                            Get Started
                        </Button>
                    </Section>
                    <Section style={{ marginTop: '40px', borderTop: '1px solid #eee', paddingTop: '20px' }}>
                        <Link href={unsubscribeUrl} style={{ color: '#666', fontSize: '12px' }}>
                            Unsubscribe from this newsletter
                        </Link>
                    </Section>
                </Container>
            </Body>
        </Html>
    );
}
```

### 3. Send the React Email

```go
func sendReactEmail() error {
    client := resend.NewClient("re_YOUR_API_KEY")
    ctx := context.Background()
    
    // Render React component to HTML (use a library like gotempl or gen)
    htmlBody := renderReactTemplate("<Newsletter title='...' content='...' unsubscribeUrl='...' />")
    
    params := &resend.SendEmailRequest{
        From:    "Aetheris Team <newsletter@aetheris.ai>",
        To:      []string{"subscriber@example.com"},
        Subject: "What's New in Aetheris",
        Html:    htmlBody,
    }
    
    _, err := client.Email.Send(ctx, params)
    return err
}
```

## Integration with Email Collection

Resend doesn't store subscribers, so combine it with:

1. **Buttondown** - Use Resend only for sending, Buttondown for subscriber management
2. **Database** - Store emails in your DB, use Resend to send
3. **ConvertKit/Mailchimp** - Store in their system, send via Resend API

## Best Practices

1. **Verify your domain** - Required for sending
2. **Use custom tracking domain** - Better deliverability
3. **Warm up your IP** - Start slow, increase volume gradually
4. **Monitor bounces** - Remove invalid emails
5. **CAN-SPAM compliance** - Always include unsubscribe link

## Resources

- [Resend Documentation](https://resend.com/docs)
- [React Email Components](https://react.email/docs/components)
- [Email Template Gallery](https://react.email/examples)

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Domain not verified | Add DNS records in Resend dashboard |
| Emails going to spam | Use custom tracking domain, warm up IP |
| Rate limits | Free tier: 100 emails/minute |
| API errors | Check API key, verify domain ownership |
