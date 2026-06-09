# Bedrock on EKS (IRSA)

Operator AI uses the AWS credential chain when Bedrock is enabled and no static keys are in `reaperc2-ai-secrets`. Without **IRSA**, pods often pick up the **node group instance role**, which typically lacks `bedrock:InvokeModel` → `AccessDeniedException`.

## Option A — IRSA (recommended)

1. Enable model access in the Bedrock console for the models you need (e.g. Claude Opus 4.7 inference profile in `us-east-1`).

2. Create an IAM policy from `bedrock-iam-policy.json` (replace `123456789012` with your account ID):

   ```bash
   aws iam create-policy \
     --policy-name ReaperC2BedrockInvoke \
     --policy-document file://examples/bedrock-iam-policy.json
   ```

3. Associate a role with the `reaperc2` ServiceAccount (replace cluster name and region):

   ```bash
   eksctl create iamserviceaccount \
     --cluster=YOUR_CLUSTER \
     --region=us-east-1 \
     --namespace=reaperc2-ns \
     --name=reaperc2 \
     --role-name reaperc2-bedrock \
     --attach-policy-arn arn:aws:iam::123456789012:policy/ReaperC2BedrockInvoke \
     --approve \
     --override-existing-serviceaccounts
   ```

   Or annotate manually after creating the role:

   ```yaml
   eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/reaperc2-bedrock
   ```

4. Ensure `reaperc2-ai-config` has `REAPER_AI_BEDROCK_USE_IAM: "1"` (set in `../operator-ai.local.yaml`).

5. Rollout:

   ```bash
   kubectl apply -k ..
   kubectl rollout restart deployment/reaperc2-deployment -n reaperc2-ns
   ```

6. Verify the pod assumes the IRSA role (not the node instance role):

   ```bash
   kubectl exec -n reaperc2-ns deployment/reaperc2-deployment -- \
     env | grep -E '^AWS_ROLE_ARN|^AWS_WEB_IDENTITY'
   ```

## Option B — Bedrock API key or IAM user keys

Apply keys via `deployments/k8s/operator-ai.yaml` (or a local patch) into `reaperc2-ai-secrets`:

- **Bedrock API key:** `REAPER_AI_BEDROCK_API_KEY`
- **IAM user:** `REAPER_AI_BEDROCK_ACCESS_KEY_ID` + `REAPER_AI_BEDROCK_SECRET_ACCESS_KEY`

Do **not** attach Bedrock permissions to the node group role unless you accept that blast radius.

## Inference profile IDs

Use inference profile IDs in `REAPER_AI_BEDROCK_MODELS`, e.g. `us.anthropic.claude-opus-4-7`. See `docs/operator-guide-ai.md`.
