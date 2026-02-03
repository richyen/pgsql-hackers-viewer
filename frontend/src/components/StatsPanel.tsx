import React from 'react';
import { Stats } from '../api/client';
import styles from './StatsPanel.module.css';

interface StatsPanelProps {
  stats: Stats | null;
  isLoading: boolean;
  onMboxSync: () => void;
  onMboxUpload: (file: File) => void;
  onReset: () => void;
}

export const StatsPanel: React.FC<StatsPanelProps> = ({
  stats,
  isLoading,
  onMboxSync,
  onMboxUpload,
  onReset,
}) => {
  const [isUploading, setIsUploading] = React.useState(false);

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setIsUploading(true);
    try {
      await onMboxUpload(file);
    } finally {
      setIsUploading(false);
      // Clear input
      e.target.value = '';
    }
  };

  if (!stats) {
    return <div className={styles.container}>Loading stats...</div>;
  }

  const statusColors: { [key: string]: string } = {
    'in-progress': '#10b981',
    discussion: '#3b82f6',
    stalled: '#f59e0b',
    abandoned: '#ef4444',
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h2>Statistics</h2>
        <div className={styles.buttonGroup}>
          <button
            className={styles.syncButton}
            onClick={onMboxSync}
            disabled={isLoading}
            title="Download and sync mbox archives from postgresql.org (last 365 days)"
          >
            {isLoading ? 'Syncing...' : 'Sync Mbox Files'}
          </button>
          <label 
            className={styles.uploadLabel}
            title="Upload a local .mbox file to import messages"
          >
            <input
              type="file"
              accept=".mbox"
              onChange={handleFileChange}
              disabled={isUploading}
              style={{ display: 'none' }}
            />
            {isUploading ? 'Uploading...' : 'Upload Mbox'}
          </label>
          <button
            className={styles.resetButton}
            onClick={onReset}
            disabled={isLoading}
            title="Clear all threads and messages; next sync will re-download from scratch"
          >
            Reset Database
          </button>
        </div>
      </div>

      <div className={styles.stats}>
        <div className={styles.stat}>
          <span className={styles.label}>Total Threads</span>
          <span className={styles.value}>{stats.total_threads}</span>
        </div>
        <div className={styles.stat}>
          <span className={styles.label}>Total Messages</span>
          <span className={styles.value}>{stats.total_messages}</span>
        </div>
        <div className={styles.stat}>
          <span className={styles.label}>Last Sync</span>
          <span className={styles.value}>
            {stats.last_sync
              ? new Date(stats.last_sync).toLocaleString()
              : 'Never'}
          </span>
        </div>
      </div>

      <div className={styles.statusBreakdown}>
        <h3>Threads by Status</h3>
        <div className={styles.statusList}>
          {Object.entries(stats.by_status).map(([status, count]) => (
            <div key={status} className={styles.statusItem}>
              <div
                className={styles.statusDot}
                style={{ backgroundColor: statusColors[status] }}
              />
              <span className={styles.statusLabel}>
                {status.charAt(0).toUpperCase() + status.slice(1).replace('-', ' ')}
              </span>
              <span className={styles.statusCount}>{count}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
