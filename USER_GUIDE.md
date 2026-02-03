# PostgreSQL Mailing List Thread Analyzer - User Guide

## Overview

This application analyzes threads from the pgsql-hackers mailing list to help you identify which threads are actively being worked on versus those that are still under discussion or have stalled.

## Thread Status Classification

Threads are automatically classified into four categories based on their activity and content:

### 🟢 In Progress
**Criteria:**
- Thread contains patch files or code contributions
- Has active code review activity
- Recent messages within the last 7 days

**What it means:** These threads have active development work happening. Patches are being submitted, reviewed, and iterated on.

### 🔵 Discussion
**Criteria:**
- Active conversation (recent messages within 7 days)
- No patches or development work submitted yet
- Multiple participants engaged in the discussion

**What it means:** These threads are in the ideation or planning phase. The community is discussing the approach, requirements, or feasibility before starting implementation.

### 🟡 Stalled
**Criteria:**
- No activity for 7-30 days
- Thread had previous activity but went quiet
- May or may not have patches

**What it means:** These threads have lost momentum. They may need someone to pick them up, or they're waiting for feedback, review, or a specific event.

### 🔴 Abandoned
**Criteria:**
- No activity for 30+ days
- No recent updates or responses

**What it means:** These threads appear to have been dropped. They might be waiting for someone to revive them, or the work may have been completed through other means.

## Features

### Sync Progress Bar
When syncing mbox archives, you'll see a progress bar showing:
- Current month being synced
- Progress: X / Y months completed
- Latest message fetched timestamp

This helps you understand:
- How much data has been synced
- How outdated the local archive is
- When the most recent message was retrieved

### Thread Links
Each message includes a "View Archive" link that takes you directly to the message on postgresql.org's official archive. This allows you to:
- See the full email formatting
- Access any attachments
- View the complete thread context on the official site

### Message Content
Click "Show Message Content" to expand and view the full text of any message inline. This lets you:
- Read the discussion without leaving the app
- Search for specific keywords
- Quickly scan message contents

## Syncing Data

### Sync Mbox Files
**Recommended method** - Downloads monthly mbox archives directly from postgresql.org.
- Initial sync: Downloads last 365 days of messages
- Incremental sync: Only downloads new months since last sync
- Requires no credentials (public archives)

### Upload Mbox
Upload a local .mbox file if you:
- Have a downloaded archive from postgresql.org
- Want to analyze a specific time period
- Are working offline

## Tips

- **Use filters** to focus on specific thread types
- **Regular syncs** keep data current (new messages arrive daily)
- **Check the progress bar** to see how up-to-date your data is
- **View archive links** for full context and attachments
- **Reset database** if sync gets corrupted or you want a fresh start

## Classification Algorithm Details

The analyzer examines each thread to determine its status by:

1. **Parsing message content** for keywords like "patch", "v2", "[PATCH]", "attached", etc.
2. **Analyzing review patterns** by looking for phrases like "looks good", "LGTM", "reviewed-by", etc.
3. **Calculating time since last activity** from message timestamps
4. **Counting unique participants** to gauge community engagement

The classification runs automatically after each sync and can be viewed in real-time through the statistics panel.
