import React, { useRef, useEffect } from 'react';
import { Thread } from '../api/client';
import styles from './ThreadList.module.css';

interface ThreadListProps {
  threads: Thread[];
  onSelectThread: (thread: Thread) => void;
  selectedStatus?: string;
  onLoadMore?: () => void;
  hasMore?: boolean;
  isLoadingMore?: boolean;
}

export const ThreadList: React.FC<ThreadListProps> = ({
  threads,
  onSelectThread,
  selectedStatus,
  onLoadMore,
  hasMore = true,
  isLoadingMore = false,
}) => {
  const listRef = useRef<HTMLUListElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!sentinelRef.current || !onLoadMore || !hasMore || isLoadingMore) {
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        const firstEntry = entries[0];
        if (firstEntry.isIntersecting && hasMore && !isLoadingMore) {
          onLoadMore();
        }
      },
      {
        root: null,
        rootMargin: '100px',
        threshold: 0.1,
      }
    );

    observer.observe(sentinelRef.current);

    return () => {
      if (sentinelRef.current) {
        observer.unobserve(sentinelRef.current);
      }
    };
  }, [onLoadMore, hasMore, isLoadingMore]);
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'in-progress':
        return '#10b981';
      case 'has-patch':
        return '#8b5cf6';
      case 'stalled-patch':
        return '#ec4899';
      case 'discussion':
        return '#3b82f6';
      case 'stalled':
        return '#f59e0b';
      case 'abandoned':
        return '#ef4444';
      default:
        return '#6b7280';
    }
  };

  return (
    <div className={styles.container}>
      <h2>Threads</h2>
      {threads.length === 0 ? (
        <p className={styles.empty}>No threads found</p>
      ) : (
        <ul className={styles.list} ref={listRef}>
          {threads.map((thread) => (
            <li
              key={thread.id}
              className={styles.item}
              onClick={() => onSelectThread(thread)}
            >
              <div className={styles.header}>
                <h3>{thread.subject}</h3>
                <span
                  className={styles.badge}
                  style={{ backgroundColor: getStatusColor(thread.status) }}
                >
                  {thread.status}
                </span>
              </div>
              <div className={styles.meta}>
                <span>{thread.message_count} messages</span>
                <span>{thread.unique_authors} authors</span>
                <span>
                  Updated: {new Date(thread.last_message_at).toLocaleDateString()}
                </span>
              </div>
            </li>
          ))}
          {hasMore && (
            <div ref={sentinelRef} className={styles.sentinel}>
              {isLoadingMore && <div className={styles.loader}>Loading more threads...</div>}
            </div>
          )}
        </ul>
      )}
    </div>
  );
};
