import React from 'react';
import { Message } from '../api/client';
import styles from './MessageThread.module.css';

interface MessageThreadProps {
  messages: Message[];
  isLoading: boolean;
}

export const MessageThread: React.FC<MessageThreadProps> = ({
  messages,
  isLoading,
}) => {
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
          {messages.map((msg) => (
            <div key={msg.id} className={styles.message}>
              <div className={styles.messageHeader}>
                <div>
                  <h4>{msg.subject}</h4>
                  <p className={styles.author}>
                    {msg.author} &lt;{msg.author_email}&gt;
                  </p>
                </div>
                <span className={styles.date}>
                  {new Date(msg.created_at).toLocaleString()}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
