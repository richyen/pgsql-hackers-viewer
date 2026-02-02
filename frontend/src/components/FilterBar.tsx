import React, { useState, useEffect } from 'react';
import styles from './FilterBar.module.css';

interface FilterBarProps {
  onStatusChange: (status: string | undefined) => void;
  selectedStatus?: string;
}

export const FilterBar: React.FC<FilterBarProps> = ({
  onStatusChange,
  selectedStatus,
}) => {
  const statuses = ['in-progress', 'discussion', 'stalled', 'abandoned'];

  return (
    <div className={styles.container}>
      <h3>Filter by Status</h3>
      <div className={styles.buttons}>
        <button
          className={`${styles.button} ${!selectedStatus ? styles.active : ''}`}
          onClick={() => onStatusChange(undefined)}
        >
          All
        </button>
        {statuses.map((status) => (
          <button
            key={status}
            className={`${styles.button} ${selectedStatus === status ? styles.active : ''}`}
            onClick={() => onStatusChange(status)}
          >
            {status.charAt(0).toUpperCase() + status.slice(1).replace('-', ' ')}
          </button>
        ))}
      </div>
    </div>
  );
};
