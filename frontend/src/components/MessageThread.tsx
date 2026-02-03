import React, { useState, useEffect, useRef } from 'react';
import { Message } from '../api/client';
import styles from './MessageThread.module.css';

interface MessageThreadProps {
  messages: Message[];
  isLoading: boolean;
  threadFirstAuthorEmail?: string;
  highlightedMessageId?: string | null;
  threadId?: string;
}

// Helper to generate postgresql.org archive link for a message
const getArchiveLink = (messageId: string): string => {
  // Extract the actual message ID from <...> format
  const cleanId = messageId.replace(/[<>]/g, '');
  return `https://www.postgresql.org/message-id/${cleanId}`;
};

// Helper to generate deep link URL for a message
const getMessageDeepLink = (threadId: string, messageId: string): string => {
  return `${window.location.origin}${window.location.pathname}?thread=${threadId}&message=${messageId}`;
};

// Helper to get patch status badge info
const getPatchStatusBadge = (status?: string) => {
  switch (status) {
    case 'committed':
      return { label: 'Committed', className: styles.statusCommitted };
    case 'accepted':
      return { label: 'Ready for Committer', className: styles.statusAccepted };
    case 'rejected':
      return { label: 'Rejected', className: styles.statusRejected };
    case 'proposed':
      return { label: 'Proposed', className: styles.statusProposed };
    default:
      return null;
  }
};

export const MessageThread: React.FC<MessageThreadProps> = ({
  messages,
  isLoading,
  threadFirstAuthorEmail,
  highlightedMessageId,
  threadId,
}) => {
  const [expandedMessages, setExpandedMessages] = useState<Set<string>>(new Set());
  const messageRefs = useRef<{ [key: string]: HTMLDivElement | null }>({});
  
  // Determine the OP email from first message if not provided
  const opEmail = threadFirstAuthorEmail || (messages.length > 0 ? messages[0].author_email : '');

  // Scroll to highlighted message when component loads or messages change
  useEffect(() => {
    if (highlightedMessageId && messageRefs.current[highlightedMessageId]) {
      setTimeout(() => {
        messageRefs.current[highlightedMessageId]?.scrollIntoView({
          behavior: 'smooth',
          block: 'center',
        });
      }, 100);
    }
  }, [highlightedMessageId, messages]);

  const toggleMessage = (id: string) => {
    setExpandedMessages(prev => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const copyDeepLink = (messageId: string) => {
    if (!threadId) return;
    const link = getMessageDeepLink(threadId, messageId);
    navigator.clipboard.writeText(link).then(() => {
      // Could add a toast notification here
      console.log('Link copied to clipboard');
    });
  };

  const jumpToNextPatch = () => {
    const patchMessages = messages.filter(msg => msg.has_patch);
    if (patchMessages.length === 0) return;
    
    const firstPatchRef = messageRefs.current[patchMessages[0].id];
    if (firstPatchRef) {
      firstPatchRef.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  };

  if (isLoading) {
    return <div className={styles.container}>Loading messages...</div>;
  }

  const patchCount = messages.filter(msg => msg.has_patch).length;

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h2>Messages</h2>
        {patchCount > 0 && (
          <div className={styles.patchInfo}>
            <span className={styles.patchCount}>
              📎 {patchCount} message{patchCount !== 1 ? 's' : ''} with patches
            </span>
            <button className={styles.jumpButton} onClick={jumpToNextPatch}>
              Jump to First Patch
            </button>
          </div>
        )}
      </div>
      {messages.length === 0 ? (
        <p className={styles.empty}>No messages found</p>
      ) : (
        <div className={styles.messages}>
          {messages.map((msg) => {
            const isExpanded = expandedMessages.has(msg.id);
            const isHighlighted = highlightedMessageId === msg.id;
            const patchBadge = msg.has_patch ? getPatchStatusBadge(msg.patch_status) : null;

            return (
              <div 
                key={msg.id} 
                className={`${styles.message} ${isHighlighted ? styles.highlighted : ''} ${msg.has_patch ? styles.hasPatch : ''}`}
                ref={(el) => (messageRefs.current[msg.id] = el)}
                id={`message-${msg.id}`}
              >
                <div className={styles.messageHeader}>
                  <div className={styles.headerLeft}>
                    <h4>{msg.subject}</h4>
                    <p className={styles.author}>
                      {msg.author} &lt;{msg.author_email}&gt;
                      {msg.author_email === opEmail && (
                        <span className={styles.opBadge} title="Original Poster">OP</span>
                      )}
                      {msg.has_patch && (
                        <span className={styles.patchBadge} title="Contains patch">
                          📎 Patch
                        </span>
                      )}
                      {patchBadge && (
                        <span className={`${styles.statusBadge} ${patchBadge.className}`} title={`Status: ${patchBadge.label}`}>
                          {patchBadge.label}
                        </span>
                      )}
                      {msg.commitfest_id && (
                        <span className={styles.commitfestBadge} title={`Commitfest ID: ${msg.commitfest_id}`}>
                          CF: {msg.commitfest_id}
                        </span>
                      )}
                    </p>
                  </div>
                  <div className={styles.headerRight}>
                    <span className={styles.date}>
                      {new Date(msg.created_at).toLocaleString()}
                    </span>
                    {threadId && (
                      <button
                        className={styles.linkButton}
                        onClick={() => copyDeepLink(msg.id)}
                        title="Copy link to this message"
                      >
                        🔗 Copy Link
                      </button>
                    )}
                    <a
                      href={getArchiveLink(msg.message_id)}
                      target="_blank"
                      rel="noopener noreferrer"
                      className={styles.archiveLink}
                      title="View on postgresql.org"
                    >
                      View Archive
                    </a>
                  </div>
                </div>
                {msg.body && (
                  <div className={styles.messageBody}>
                    <button 
                      className={styles.toggleButton}
                      onClick={() => toggleMessage(msg.id)}
                    >
                      {isExpanded ? '▼ Hide' : '▶ Show'} Message Content
                    </button>
                    {isExpanded && (
                      <pre className={styles.bodyContent}>{msg.body}</pre>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
