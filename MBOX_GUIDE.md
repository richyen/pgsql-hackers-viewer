# Mbox File Support

The PostgreSQL Mailing List Thread Analyzer supports importing messages from mbox files, which is a standard mail archive format used for storing email messages.

## What is Mbox?

An mbox file is a plain text file containing email messages separated by lines beginning with "From ". This is the standard format used by:
- Local mail clients (Thunderbird, Evolution, etc.)
- Mail server backups
- Mailing list archives

## Getting Mbox Files

### Option 1: Download from PostgreSQL Archives

The PostgreSQL mailing list maintains complete archives in various formats:

1. Visit: https://www.postgresql.org/list/pgsql-hackers/
2. Look for archive download options
3. Download the mbox files you want to analyze

### Option 2: Export from Your Email Client

#### Gmail (via IMAP)
- Use a mail client or script to export via IMAP
- Or use Gmail's "Download" feature for labels

#### Thunderbird
1. Right-click on folder → Properties → Repair Folder
2. Export using an add-on like "ImportExportTools NG"

#### Other Clients
- Check your email client's export/backup features
- Usually under File → Export or Backup

### Option 3: Generate from Python/Scripts

```python
import imaplib
import mailbox

# Connect to IMAP server
imap = imaplib.IMAP4_SSL('imap.gmail.com')
imap.login('email@gmail.com', 'app-password')

# Create mbox file
mbox = mailbox.mbox('pgsql-hackers.mbox')

# Fetch messages
imap.select('pgsql-hackers')
status, messages = imap.search(None, 'ALL')

for msg_id in messages[0].split():
    status, msg_data = imap.fetch(msg_id, '(RFC822)')
    mbox.add(msg_data[0][1])

mbox.close()
imap.close()
```

## Uploading Mbox Files

### Via Web UI

1. Click **"Upload Mbox"** button in the Statistics panel
2. Select your mbox file (`.mbox` extension recommended)
3. The file will be uploaded and messages imported automatically

### Via API

```bash
curl -X POST http://localhost:8080/api/sync/mbox \
  -F "file=@/path/to/pgsql-hackers.mbox"
```

## Syncing All Stored Mbox Files

If you've added multiple mbox files to the `data/` directory, sync them all:

### Via Web UI

Click **"Sync Mbox Files"** button

### Via API

```bash
curl -X POST http://localhost:8080/api/sync/mbox/all
```

## File Storage

Mbox files are stored in the `data/` directory:

```
data/
├── pgsql-hackers-2024-01.mbox
├── pgsql-hackers-2024-02.mbox
├── pgsql-hackers-2024-03.mbox
└── ...
```

You can also manually place mbox files in this directory and run the sync.

## Directory Structure

```
backend/data/          # Mbox files stored here
.gitignore            # Excludes data/ from git
```

## Mbox File Format

A typical mbox file looks like:

```
From sender@example.com Fri Feb 02 12:00:00 2024
Subject: Thread title
From: John Doe <john@example.com>
Date: Fri, 02 Feb 2024 12:00:00 +0000
Message-ID: <message-id@example.com>

Message body content here...

From another@example.com Fri Feb 02 13:00:00 2024
Subject: RE: Thread title
From: Jane Smith <jane@example.com>
Date: Fri, 02 Feb 2024 13:00:00 +0000
Message-ID: <reply-message-id@example.com>

Reply message body...
```

## Mbox Parser Features

The parser supports:

- Standard mbox format with "From " delimiters
- RFC2822 email headers (Subject, From, Date, Message-ID)
- Multiline message bodies
- Various date formats
- Email addresses in multiple formats:
  - `name <email@example.com>`
  - `email@example.com`

## Processing

When you upload or sync mbox files:

1. **Parsing**: Messages are extracted from mbox format
2. **Grouping**: Messages are grouped by subject into threads
3. **Storage**: Threads and messages are stored in PostgreSQL
4. **Analysis**: Threads are automatically classified by activity status
5. **Indexing**: Database indexes are used for fast queries

## Database Storage

Messages from mbox files are stored in the same tables as IMAP messages:

- `threads` - Thread metadata
- `messages` - Individual email messages
- `thread_activities` - Activity metrics

## Duplicate Handling

The system prevents duplicate imports:

- Each message is identified by `message_id` header
- Duplicate message IDs are skipped on import
- Re-running sync with same mbox file won't create duplicates

## Tips for Large Archives

For very large mbox files (100MB+):

1. **Split files**: Use `mboxsplit` tool to split into smaller files
2. **Batch uploads**: Upload multiple smaller files
3. **Check storage**: Ensure you have disk space for the data directory

## Troubleshooting

### Upload fails

- Check file is valid mbox format
- Ensure it ends with `.mbox` extension
- Check disk space
- View logs: `docker-compose logs backend`

### Messages not appearing

1. Verify mbox file format is correct
2. Check message date headers are parseable
3. Check database for errors: `SELECT COUNT(*) FROM messages;`
4. Try re-uploading with smaller file

### Missing message headers

If the parser skips some messages:
- Check message-id header exists
- Verify subject line is present
- Check date format is RFC2822 compatible

## Combining IMAP and Mbox

You can use both IMAP syncing and mbox file uploads simultaneously:

- IMAP for real-time updates from active lists
- Mbox files for historical archive data

The system will handle duplicates automatically.

## Export from Mbox

Currently, export of mbox format is not supported. However, you can:

- Export threads as CSV/JSON via API (future feature)
- Use database queries to extract thread data
- Connect third-party tools directly to PostgreSQL
