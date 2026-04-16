# Mailchimp Integration Guide

This guide explains how to integrate the Aetheris newsletter with Mailchimp.

## Prerequisites

- Mailchimp account (free tier available)
- API key from Mailchimp dashboard

## Option 1: Embedded Form (Simplest)

### 1. Create an Embedded Form in Mailchimp

1. Go to **Audience** → **Signup forms** → **Embedded forms**
2. Customize your form fields
3. Copy the generated HTML form code
4. Replace the form in `subscribe.html` with your Mailchimp form

### 2. Update subscribe.html

Replace the `<form>` section with your Mailchimp embed code:

```html
<!-- Mailchimp Embedded Form - Replace with your actual form action -->
<form 
    action="https://YOUR_ACCOUNT.us1.list-manage.com/subscribe/post?u=XXXX&id=XXXX" 
    method="post"
    id="mc-embedded-subscribe-form"
    name="mc-embedded-subscribe-form"
>
    <input type="email" name="EMAIL" placeholder="your.email@example.com" required>
    <button type="submit">Subscribe to Updates</button>
</form>
```

## Option 2: Mailchimp API (Programmatic)

### 1. Install Mailchimp SDK

```bash
go get github.com/go-mailchimp/mailchimp
```

### 2. Add Subscriber via API

```go
package main

import (
    "context"
    "fmt"
    mc "github.com/go-mailchimp/mailchimp"
)

func addSubscriber(email, firstName, lastName string) error {
    client := mc.NewMailchimp("YOUR_API_KEY")
    
    ctx := context.Background()
    
    // Create subscriber
    _, err := client.Lists.AddListMember(ctx, "YOUR_LIST_ID", mc.ListMember{
        EmailAddress: email,
        Status:       "pending", // Sends confirmation email
        MergeFields: map[string]interface{}{
            "FNAME": firstName,
            "LNAME": lastName,
        },
    })
    
    if err != nil {
        return fmt.Errorf("failed to add subscriber: %w", err)
    }
    
    return nil
}
```

### 3. Handle Webhook for Subscribe Events

Mailchimp can send webhook notifications when subscribers join:

```go
// In your HTTP handler (using Hertz)
func HandleMailchimpWebhook(ctx context.Context, c *app.RequestContext) {
    type webhookPayload struct {
        Type   string `json:"type"`
        Email  string `json:"email"`
        FName  string `json:"fName"`
        LName  string `json:"lName"`
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

## Option 3: Zapier/Make Integration

For no-code integration:

1. Create a **Webhooks** → **Catch Hook** zap in Zapier
2. Copy the webhook URL
3. In Mailchimp, go to **Audience** → **Signup forms** → **Form builder**
4. Add a hidden field with the webhook URL
5. Configure Mailchimp to POST to that URL on form submission

## Managing Subscribers

### Export Subscriber List

```bash
# Using Mailchimp API
curl -X GET "https://us1.api.mailchimp.com/3.0/lists/YOUR_LIST_ID/members" \
  -H "Authorization: apikey YOUR_API_KEY"
```

### Tag Subscribers

```go
// Add tag to subscriber
client.Lists.UpdateListMemberTags(ctx, "YOUR_LIST_ID", "subscriber_email_hash", mc.ListMemberTags{
    Tags: []mc.ListMemberTag{
        {Name: "aetheris-users", Status: "active"},
    },
})
```

## Best Practices

1. **Use "pending" status** to send confirmation emails (CAN-SPAM compliance)
2. **Tag subscribers** to segment your audience (e.g., "aetheris-users", "beta-testers")
3. **Handle unsubscribes** via webhooks to keep your system in sync
4. **Rate limiting**: Mailchimp limits API calls; implement retry logic

## Resources

- [Mailchimp API Documentation](https://mailchimp.com/developer/marketing/api/)
- [Embedded Form Builder](https://mailchimp.com/help/create-embedded-signup-forms/)
- [Webhook Documentation](https://mailchimp.com/developer/marketing/api/list-webhooks/)

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Form not submitting | Check your Mailchimp form action URL |
| Double opt-in not working | Ensure status is "pending" not "subscribed" |
| API rate limits | Implement exponential backoff |
| Webhook not firing | Verify webhook URL is publicly accessible |
