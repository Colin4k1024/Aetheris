# M4 Signature Guide - 数字签名机制

## 概述

Aetheris 3.0-M4 引入 Ed25519 数字签名，为证据包提供不可否认性，确保证据包由可信组织签名且未被篡改。

---

## 核心概念

### 数字签名

使用 Ed25519 算法：
- **高性能**: 签名/验证速度快
- **安全**: 256-bit 安全级别
- **紧凑**: 签名只有 64 bytes
- **确定性**: 相同数据相同签名

### 签名格式

```
ed25519:<key_id>:<signature_base64>
```

示例：
```
ed25519:org_primary_key:SGVsbG8gV29ybGQhIFRoaXMgaXMgYSBzaWdu...
```

---

## Current Integrated Path

The integrated path signs evidence ZIPs during export when API config enables `security.evidence_signing`. See [evidence-signing.md](evidence-signing.md).

### Configure signing

```yaml
security:
  evidence_signing:
    enabled: true
    key_id: "org_primary_key"
    private_key_base64: "${AETHERIS_EVIDENCE_SIGNING_PRIVATE_KEY}"
    public_key_base64: "${AETHERIS_EVIDENCE_SIGNING_PUBLIC_KEY}"
```

### Export signed evidence

```bash
aetheris export <job_id> --output evidence.zip
```

### Verify signature

```bash
aetheris verify evidence.zip --public-key <base64-public-key>
```

The CLI verifies file hashes, event hash chain, ledger consistency, and the Ed25519 signature when a public key is supplied.

---

## 密钥管理

### 存储方式

Current implementation reads raw base64 Ed25519 keys from config/env. KMS/Vault-backed custody is a production-readiness follow-up, not part of the integrated slice yet.

### 密钥轮换

```bash
# 1. Generate and distribute a new Ed25519 key pair out of band.
# 2. Update API config/env.
vi configs/api.yaml
# security.evidence_signing.key_id: "org_key_2026"

# 3. Keep old public keys for historical evidence verification.
```

---

## 安全最佳实践

1. **私钥保护**:
   - 永不记录到日志
   - 永不包含在证据包中
   - 使用 HSM/KMS（生产环境）

2. **密钥轮换**:
   - 每年轮换一次
   - 保留旧公钥（验证历史）
   - 逐步淘汰过期密钥

3. **访问控制**:
   - 只有 Admin 角色可签名
   - 签名操作记录审计日志
   - 多重签名（关键操作）

---

## 故障排查

### 问题: 签名验证失败

**原因**: 公钥不匹配或证据包被篡改

**解决**:
1. 检查公钥是否正确
2. 验证证据包完整性
3. 查看审计日志

### 问题: 私钥不可用

**原因**: Vault 连接失败或密钥被删除

**解决**:
1. 检查 Vault 连接
2. 恢复密钥备份
3. 生成新密钥（标记为新 key_id）

---

## 下一步

- 查看 `docs/m4-distributed-ledger-guide.md` 了解跨组织验证
- 查看 `docs/aetheris-3.0-complete.md` 了解完整能力
