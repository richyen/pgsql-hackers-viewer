import React from 'react';
import styles from './FilterBar.module.css';

interface FilterBarProps {
  onStatusChange: (status: string | undefined) => void;
  selectedStatus?: string;
  onSearchChange: (search: string) => void;
  searchTerm: string;
}

export const FilterBar: React.FC<FilterBarProps> = ({
  onStatusChange,
  selectedStatus,
  onSearchChange,
  searchTerm,
}) => {
  const statuses = [
    { value: 'in-progress', label: 'In Progress', tooltip: 'Threads with patches and active review activity', color: '#10b981' },
    { value: 'has-patch', label: 'Has Patch', tooltip: 'Threads with patches but not yet actively reviewed', color: '#8b5cf6' },
    { value: 'stalled-patch', label: 'Stalled Patch', tooltip: 'Patches without acceptance/review for 14+ days', color: '#ec4899' },
    { value: 'discussion', label: 'Discussion', tooltip: 'Active threads without development work yet', color: '#3b82f6' },
    { value: 'stalled', label: 'Stalled', tooltip: 'No activity for 7-30 days', color: '#f59e0b' },
    { value: 'abandoned', label: 'Abandoned', tooltip: 'No activity for 30+ days', color: '#ef4444' },
  ];

  return (
    <div className={styles.container}>
      <div className={styles.searchSection}>
        <h3>Search</h3>
        <input
          type="text"
          className={styles.searchInput}
          placeholder="Search by subject or Message-ID..."
          value={searchTerm}
          onChange={(e) => onSearchChange(e.target.value)}
        />
      </div>
      <div className={styles.statusSection}>
        <h3>Filter by Status</h3>
        <div className={styles.buttons}>
        <button
          className={`${styles.button} ${!selectedStatus ? styles.active : ''}`}
          onClick={() => onStatusChange(undefined)}
          title="Show all threads regardless of status"
        >
          All
        </button>
        {statuses.map((status) => (
          <button
            key={status.value}
            className={`${styles.button} ${selectedStatus === status.value ? styles.active : ''}`}
            onClick={() => onStatusChange(status.value)}
            title={status.tooltip}
            style={selectedStatus === status.value ? {
              backgroundColor: status.color,
              borderColor: status.color,
              color: 'white'
            } : {}}
          >
            {status.label}
          </button>
        ))}
        </div>
      </div>
    </div>
  );
};
