import React, { useEffect, useState } from 'react';
import { threadAPI, SyncProgress } from '../api/client';
import styles from './SyncProgressBar.module.css';

export const SyncProgressBar: React.FC = () => {
  const [progress, setProgress] = useState<SyncProgress | null>(null);

  useEffect(() => {
    const fetchProgress = async () => {
      try {
        const response = await threadAPI.getSyncProgress();
        setProgress(response.data);
      } catch (error) {
        console.error('Error fetching sync progress:', error);
      }
    };

    fetchProgress();
    const interval = setInterval(fetchProgress, 2000); // Poll every 2 seconds

    return () => clearInterval(interval);
  }, []);

  if (!progress) return null;

  const percentage = progress.total_months > 0
    ? Math.round((progress.months_synced / progress.total_months) * 100)
    : 0;

  const isActive = progress.is_syncing;

  return (
    <div className={styles.progressContainer}>
      {isActive && (
        <>
          <div className={styles.progressInfo}>
            <span className={styles.progressLabel}>
              Syncing: {progress.current_month || 'Initializing...'}
            </span>
            <span className={styles.progressStats}>
              {progress.months_synced} / {progress.total_months} months
            </span>
          </div>
          <div className={styles.progressBar}>
            <div 
              className={styles.progressFill}
              style={{ width: `${percentage}%` }}
            />
          </div>
        </>
      )}
      {progress.latest_message_date && (
        <div className={styles.latestMessage}>
          <span className={styles.label}>Latest message:</span>
          <span className={styles.date}>
            {new Date(progress.latest_message_date).toLocaleString()}
          </span>
        </div>
      )}
    </div>
  );
};
