# Chat

**Path:** `/chat`  
**Requires:** Active engagement

Operator-only chat for the engagement. Messages are stored in MongoDB `operator_chat`. The room key comes from the engagement’s **Slack / Discord room** field, or a stable internal id if unset.

The log **polls** every few seconds; new messages appear after **Send** (`POST /api/chat/messages`). Chat is included in **All logs** JSON / Ghostwriter exports, not in **Reports** JSON.
