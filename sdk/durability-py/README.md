# Aetheris Durability SDK (Python)

**为任何 Python agent 添加 crash recovery、checkpoint 和幂等性。零框架依赖。**

## 安装

```bash
pip install aetheris-durability

# 带 PostgreSQL 支持
pip install aetheris-durability[postgres]
```

## 30 秒上手

```python
from aetheris_durability import Runner, MemoryStore, Step

# 1. 创建 store（开发用内存，生产用 PostgresStore）
store = MemoryStore()

# 2. 创建 runner
runner = Runner(store)

# 3. 启动一个 job
job = runner.start("process-order", {
    "order_id": "ORD-123",
    "amount": 99.99,
})

# 4. 定义 steps
def validate(state):
    print(f"Validating order {state['order_id']}...")
    return {**state, "validated": True}

def charge(state):
    print(f"Charging ${state['amount']}...")
    return {**state, "charged": True, "txn_id": "TXN-456"}

def ship(state):
    print(f"Creating shipment for {state['txn_id']}...")
    return {**state, "shipped": True}

steps = [
    Step("validate", validate),
    Step("charge", charge, max_retries=3),  # 失败自动重试
    Step("ship", ship),
]

# 5. 执行 — 如果中间挂了，再调一次 execute 就从上次继续
result = runner.execute(job.id, steps)
print(result)
```

## Crash Recovery

```python
# 假设 process 在 "charge" 步骤挂了
try:
    runner.execute(job.id, steps)
except RuntimeError:
    print("Process crashed!")

# 重启后再调一次 — "validate" 被跳过（已 checkpoint），从 "charge" 继续
result = runner.execute(job.id, steps)
# 成功！validate 没有被重复执行
```

## 幂等工具（防止副作用重复执行）

```python
from aetheris_durability import IdempotentTool

idem = IdempotentTool(store)

@idem.wrap("send-email")
def send_email(input_data):
    # 这个函数每个 idempotency key 只会执行一次
    return email_service.send(input_data["to"], input_data["subject"])

# 同一个 key 只会执行一次
result1 = send_email({"to": "user@example.com"}, idempotency_key="order-123")
result2 = send_email({"to": "user@example.com"}, idempotency_key="order-123")
# result2 直接返回缓存结果，不会重复发送邮件
```

## 与 Go SDK 的对应关系

| Python | Go | 用途 |
|--------|-----|------|
| `MemoryStore` | `core.NewMemoryStore()` | 开发/测试 |
| `PostgresStore` | `postgres.NewStore()` | 生产环境 |
| `Runner` | `core.NewRunner()` | 执行 + checkpoint |
| `Step` | `core.Step{}` | 定义步骤 |
| `IdempotentTool` | `idempotent.New()` | 幂等包装器 |

## License

Apache License 2.0
