# Enterprise Integrations

> **Version**: v2.3.0+

Aetheris provides enterprise integration capabilities including LDAP/AD, message queues, and cloud storage.

## LDAP/Active Directory Integration

### Configuration

```yaml
security:
  ldap:
    url: ldaps://ldap.example.com:636
    base_dn: dc=example,dc=com
    bind_dn: cn=admin,dc=example,dc=com
    bind_password: ${LDAP_PASSWORD}
    user_filter: "(uid=%s)"        # For LDAP
    # user_filter: "(sAMAccountName=%s)"  # For Active Directory
    group_filter: "(member=%s)"
    use_ssl: true
    skip_verify: false
```

### Usage

```go
import "rag-platform/pkg/integration/ldap"

store, err := ldap.NewLDAPStore(&ldap.Config{
    URL:       "ldaps://ldap.example.com:636",
    BaseDN:    "dc=example,dc=com",
    BindDN:    "cn=admin,dc=example,dc=com",
    UserFilter: "(uid=%s)",
})

// Authenticate user
user, err := store.Authenticate(ctx, "username", "password")

// Get user groups
groups, err := store.GetUserGroups(ctx, "username")
```

### Features

- User authentication
- Group membership lookup
- Search users/groups
- AD and LDAP support

## Message Queue Integration

### Supported Providers

| Provider | Type | Use Case |
|----------|------|---------|
| Amazon SQS | Cloud | AWS workloads |
| RabbitMQ | Self-hosted | On-prem |

### SQS Configuration

```yaml
integration:
  queue:
    provider: sqs
    region: us-east-1
    access_key: ${AWS_ACCESS_KEY}
    secret_key: ${AWS_SECRET_KEY}
    queue_prefix: aetheris-
```

### RabbitMQ Configuration

```yaml
integration:
  queue:
    provider: rabbitmq
    endpoint: amqp://guest:guest@localhost:5672/
    queue_prefix: aetheris-
```

### Usage

```go
import "rag-platform/pkg/integration/queue"

// Create queue
q, err := queue.NewMessageQueue(queue.Config{
    Provider:    "sqs",
    Region:      "us-east-1",
    AccessKey:   "...",
    SecretKey:   "...",
})

// Send message
err := q.Send(ctx, "my-queue", []byte(`{"job_id": "123"}`))

// Receive messages
msgs, err := q.Receive(ctx, "my-queue", 10)
```

## Cloud Storage Integration

### Supported Providers

| Provider | Type | Use Case |
|----------|------|---------|
| Amazon S3 | Cloud | AWS workloads |
| Google GCS | Cloud | GCP workloads |
| Azure Blob | Cloud | Azure workloads |

### S3 Configuration

```yaml
integration:
  storage:
    provider: s3
    region: us-east-1
    bucket: aetheris-data
    access_key: ${AWS_ACCESS_KEY}
    secret_key: ${AWS_SECRET_KEY}
```

### Usage

```go
import "rag-platform/pkg/integration/storage"

// Create storage
s, err := storage.NewObjectStore(storage.Config{
    Provider:   "s3",
    Region:     "us-east-1",
    Bucket:     "aetheris-data",
    AccessKey:  "...",
    SecretKey:  "...",
})

// Upload
err := s.Put(ctx, "bucket", "path/to/file.txt", strings.NewReader("content"), "text/plain")

// Download
reader, err := s.Get(ctx, "bucket", "path/to/file.txt")
defer reader.Close()
```

## Security Considerations

### LDAP/AD

- Use SSL/TLS (ldaps://)
- Store credentials in secrets manager
- Regular password rotation

### Message Queues

- Enable IAM policies for SQS
- Use VPC endpoints for private access
- Encrypt messages at rest

### Cloud Storage

- Enable server-side encryption
- Use bucket policies for access control
- Enable versioning for critical data
