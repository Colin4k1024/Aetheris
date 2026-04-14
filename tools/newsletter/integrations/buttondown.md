# Buttondown Integration Guide

Buttondown is a simple, developer-friendly newsletter platform with a clean API.

## Prerequisites

- Buttondown account (($5/month for 500 subscribers))
- API token from Settings → Developer

## Option 1: Embedded Form (Simplest)

### 1. Get Your Form Embed Code

1. Go to **Settings** → **Audience** → **Embeddable form**
2. Copy the form HTML

### 2. Update subscribe.html

```html
<!-- Replace the form section with Buttondown embed -->
<form 
    action="https://buttondown.email/YOUR_USERNAME" 
    method="post"
>
    <input type="email" name="email" placeholder="your.email@example.com" required>
    <button type="submit">Subscribe</button>
</form>
```

## Option 2: Buttondown API

### 1. Get Your API Token

1. Go to **Settings** → **Developer** → **API token**
2. Save the token securely

### 2. Install HTTP Client

```bash
# No special SDK needed - just use net/http
```

### 3. Add Subscriber via API

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type Subscriber struct {
    Email string   `json:"email"`
    Tags  []string `json:"tags"`
}

func addSubscriber(apiToken, email string) error {
    payload := Subscriber{
        Email: email,
        Tags:  []string{"aetheris-users"},
    }
    
    jsonData, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal: %w", err)
    }
    
    req, err := http.NewRequest("POST", "https://api.buttondown.email/v1/subscribers", bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }
    
    req.Header.Set("Authorization", "Token "+apiToken)
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }
    
    return nil
}
```

### 4. Send Newsletter via API

```go
func sendNewsletter(apiToken, subject, body string) error {
    payload := map[string]string{
        "subject": subject,
        "body":    body,
    }
    
    jsonData, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    
    req, err := http.NewRequest("POST", "https://api.buttondown.email/v1/emails", bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    
    req.Header.Set("Authorization", "Token "+apiToken)
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}
```

### 5. Schedule Newsletter

```go
func scheduleNewsletter(apiToken, subject, body, scheduledTime string) error {
    payload := map[string]string{
        "subject":         subject,
        "body":            body,
        "scheduled_time":  scheduledTime, // ISO 8601 format: "2024-12-25T10:00:00Z"
    }
    
    jsonData, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    
    req, err := http.NewRequest("POST", "https://api.buttondown.email/v1/emails", bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    
    req.Header.Set("Authorization", "Token "+apiToken)
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}
```

## Option 3: Webhook Integration

Receive notifications when subscribers join/leave:

```go
// In your HTTP handler
func HandleButtondownWebhook(ctx context.Context, c *app.RequestContext) {
    type webhookPayload struct {
        Type  string `json:"type"` // "subscribe" or "unsubscribe"
        Email string `json:"email_address"`
    }
    
    var payload webhookPayload
    if err := c.BindJSON(&payload); err != nil {
        c.JSON(400, map[string]string{"error": "invalid payload"})
        return
    }
    
    switch payload.Type {
    case "subscribe":
        // Handle new subscriber
        fmt.Printf("New subscriber: %s\n", payload.Email)
    case "unsubscribe":
        // Handle unsubscribe
    }
}
```

Configure webhook URL in **Settings** → **Webhooks**.

## Managing Subscribers

### List All Subscribers

```bash
curl -X GET "https://api.buttondown.email/v1/subscribers" \
  -H "Authorization: Token YOUR_API_TOKEN"
```

### Remove Subscriber

```bash
curl -X DELETE "https://api.buttondown.email/v1/subscribers/EMAIL_HASH" \
  -H "Authorization: Token YOUR_API_TOKEN"
```

## Email Templates

Buttondown supports Markdown email templates. Example:

```markdown
# {{ title }}

{{ body }}

---
        
Unsubscribe: {{ unsubscribe_url }}
```

## Best Practices

1. **Use tags** to segment subscribers (e.g., "aetheris-users", "beta", "chinese-speakers")
2. **Schedule newsletters** for optimal send times (Tuesday/Thursday mornings work well)
3. **Clean up inactive subscribers** periodically

## Resources

- [Buttondown API Documentation](https://api.buttondown.email)
- [Webhook Events](https://api.buttondown.email/#webhooks)

## Comparison with Alternatives

| Feature | Buttondown | Mailchimp |
|---------|------------|-----------|
| Price | $5/mo | Free up to 500 |
| API | Simple, clean | Complex |
| Developer experience | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| Email templates | Markdown | Drag-and-drop |
| Segmentation | Tags only | Advanced |

## Troubleshooting

| Issue | Solution |
|-------|----------|
| API returns 401 | Check your API token |
| Double opt-in | Buttondown sends confirmation by default |
| Rate limits | No explicit limits; be reasonable |
| Email deliverability | Use a custom sending domain |
