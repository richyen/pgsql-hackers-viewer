import React from 'react';
import styles from './FilterBar.module.css';

interface FilterBarProps {
  onStatusChange: (status: string | undefined) => void;
  selectedStatus?: string;
}

export const FilterBar: React.FC<FilterBarProps> = ({
  onStatusChange,
  selectedStatus,
}) => {
  const statuses = [
    { value: 'in-progress', label: 'In Progress', tooltip: 'Threads with patches and active review activity', color: '#10b981' },
    { value: 'discussion', label: 'Discussion', tooltip: 'Active threads without development work yet', color: '#3b82f6' },
    { value: 'stalled', label: 'Stalled', tooltip: 'No activity for 7-30 days', color: '#f59e0b' },
    { value: 'abandoned', label: 'Abandoned', tooltip: 'No activity for 30+ days', color: '#ef4444' },
  ];

  return (
    <div className={styles.container}>
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
  );
};
