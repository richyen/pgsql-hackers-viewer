import React, { useState } from 'react';
import { Message } from '../api/client';
import styles from './MessageThread.module.css';

interface MessageThreadProps {
  messages: Message[];
  isLoading: boolean;
  threadFirstAuthorEmail?: string;
}

// Helper to generate postgresql.org archive link for a message
const getArchiveLink = (messageId: string): string => {
  // Extract the actual message ID from <...> format
  const cleanId = messageId.replace(/[<>]/g, '');
  return `https://www.postgresql.org/message-id/${cleanId}`;
};

export const MessageThread: React.FC<MessageThreadProps> = ({
  messages,
  isLoading,
  threadFirstAuthorEmail,
}) => {
  const [expandedMessages, setExpandedMessages] = useState<Set<string>>(new Set());
  
  // Determine the OP email from first message if not provided
  const opEmail = threadFirstAuthorEmail || (messages.length > 0 ? messages[0].author_email : '');

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

  if (isLoading) {
    return <div className={styles.container}>Loading messages...</div>;
  }

  return (
    <div className={styles.container}>
      <h2>Messages</h2>
      {messages.length === 0 ? (
        <p className={styles.empty}>No messages found</p>
      ) : (
        <div className={styles.messages}>
          {messages.map((msg) => {
            const isExpanded = expandedMessages.has(msg.id);
            return (
              <div key={msg.id} className={styles.message}>
                <div className={styles.messageHeader}>
                  <div className={styles.headerLeft}>
                    <h4>{msg.subject}</h4>
                    <p className={styles.author}>
                      {msg.author} &lt;{msg.author_email}&gt;
                      {msg.author_email === opEmail && (
                        <span className={styles.opBadge} title="Original Poster">OP</span>
                      )}
                    </p>
                  </div>
                  <div className={styles.headerRight}>
                    <span className={styles.date}>
                      {new Date(msg.created_at).toLocaleString()}
                    </span>
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
