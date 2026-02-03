import React from 'react';
import styles from './HelpModal.module.css';

interface HelpModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const HelpModal: React.FC<HelpModalProps> = ({ isOpen, onClose }) => {
  if (!isOpen) return null;

  return (
    <div className={styles.overlay} onClick={onClose}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <div className={styles.header}>
          <h2>Help & Classification Guide</h2>
          <button className={styles.closeButton} onClick={onClose}>
            ✕
          </button>
        </div>
        <div className={styles.content}>
          <section className={styles.section}>
            <h3>Thread Status Classification</h3>
            <p>
              Threads are automatically classified into four categories based on their activity and content:
            </p>
          </section>

          <section className={styles.section}>
            <div className={styles.statusCard}>
              <div className={styles.statusHeader}>
                <span className={styles.statusDot} style={{ backgroundColor: '#10b981' }} />
                <h4>In Progress</h4>
              </div>
              <p>
                <strong>Criteria:</strong> Contains patches or code, has active review activity, recent messages within 7 days.
              </p>
              <p>
                <strong>Meaning:</strong> Active development work with patches being submitted and reviewed.
              </p>
            </div>

            <div className={styles.statusCard}>
              <div className={styles.statusHeader}>
                <span className={styles.statusDot} style={{ backgroundColor: '#3b82f6' }} />
                <h4>Discussion</h4>
              </div>
              <p>
                <strong>Criteria:</strong> Recent activity (within 7 days), no patches yet, multiple participants.
              </p>
              <p>
                <strong>Meaning:</strong> In ideation/planning phase before implementation starts.
              </p>
            </div>

            <div className={styles.statusCard}>
              <div className={styles.statusHeader}>
                <span className={styles.statusDot} style={{ backgroundColor: '#f59e0b' }} />
                <h4>Stalled</h4>
              </div>
              <p>
                <strong>Criteria:</strong> No activity for 7-30 days.
              </p>
              <p>
                <strong>Meaning:</strong> Lost momentum, may need someone to pick it up or waiting for feedback.
              </p>
            </div>

            <div className={styles.statusCard}>
              <div className={styles.statusHeader}>
                <span className={styles.statusDot} style={{ backgroundColor: '#ef4444' }} />
                <h4>Abandoned</h4>
              </div>
              <p>
                <strong>Criteria:</strong> No activity for 30+ days.
              </p>
              <p>
                <strong>Meaning:</strong> Appears dropped, may need revival or was completed elsewhere.
              </p>
            </div>
          </section>

          <section className={styles.section}>
            <h3>Classification Algorithm</h3>
            <p>The analyzer determines thread status by:</p>
            <ul>
              <li>Parsing message content for keywords (patch, v2, [PATCH], attached, etc.)</li>
              <li>Analyzing review patterns (looks good, LGTM, reviewed-by, etc.)</li>
              <li>Calculating time since last activity from timestamps</li>
              <li>Counting unique participants for engagement metrics</li>
            </ul>
          </section>

          <section className={styles.section}>
            <h3>Features</h3>
            <dl className={styles.featureList}>
              <dt>Sync Progress Bar</dt>
              <dd>Shows current month being synced, progress, and latest message timestamp</dd>
              
              <dt>Thread Links</dt>
              <dd>Click "View Archive" to see messages on postgresql.org</dd>
              
              <dt>Message Content</dt>
              <dd>Expand messages to read full content inline</dd>
              
              <dt>Filters</dt>
              <dd>Focus on specific thread status types</dd>
            </dl>
          </section>

          <section className={styles.section}>
            <h3>Syncing Data</h3>
            <dl className={styles.featureList}>
              <dt>Sync Mbox Files (Recommended)</dt>
              <dd>Downloads archives from postgresql.org. Initial: 365 days, then incremental.</dd>
              
              <dt>Upload Mbox</dt>
              <dd>Import a local .mbox file for offline analysis</dd>
              
              <dt>Reset Database</dt>
              <dd>Clear all data; next sync re-downloads everything</dd>
            </dl>
          </section>
        </div>
      </div>
    </div>
  );
};
