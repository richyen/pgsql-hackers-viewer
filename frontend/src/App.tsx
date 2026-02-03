import React, { useState, useEffect } from 'react';
import { threadAPI, Thread, Message, Stats } from './api/client';
import { ThreadList } from './components/ThreadList';
import { MessageThread } from './components/MessageThread';
import { StatsPanel } from './components/StatsPanel';
import { FilterBar } from './components/FilterBar';
import { SyncProgressBar } from './components/SyncProgressBar';
import { HelpModal } from './components/HelpModal';
import styles from './App.module.css';

function App() {
  const [threads, setThreads] = useState<Thread[]>([]);
  const [selectedThread, setSelectedThread] = useState<Thread | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [selectedStatus, setSelectedStatus] = useState<string | undefined>();
  const [isLoading, setIsLoading] = useState(false);
  const [isMessagesLoading, setIsMessagesLoading] = useState(false);
  const [isHelpOpen, setIsHelpOpen] = useState(false);

  // Fetch stats on mount and periodically
  useEffect(() => {
    const fetchStats = async () => {
      try {
        const response = await threadAPI.getStats();
        setStats(response.data);
      } catch (error) {
        console.error('Error fetching stats:', error);
      }
    };

    fetchStats();
    const interval = setInterval(fetchStats, 30000); // Refresh every 30s

    return () => clearInterval(interval);
  }, []);

  // Fetch threads when status filter changes
  useEffect(() => {
    const fetchThreads = async () => {
      setIsLoading(true);
      try {
        const response = await threadAPI.getThreads(selectedStatus);
        setThreads(response.data || []);
        setSelectedThread(null);
        setMessages([]);
      } catch (error) {
        console.error('Error fetching threads:', error);
      } finally {
        setIsLoading(false);
      }
    };

    fetchThreads();
  }, [selectedStatus]);

  // Fetch messages when thread is selected
  useEffect(() => {
    if (!selectedThread) {
      setMessages([]);
      return;
    }

    const fetchMessages = async () => {
      setIsMessagesLoading(true);
      try {
        const response = await threadAPI.getThreadMessages(selectedThread.id);
        setMessages(response.data || []);
      } catch (error) {
        console.error('Error fetching messages:', error);
      } finally {
        setIsMessagesLoading(false);
      }
    };

    fetchMessages();
  }, [selectedThread]);

  const handleMboxSync = async () => {
    setIsLoading(true);
    try {
      await threadAPI.syncMbox();
      // Refresh data after sync
      setTimeout(async () => {
        const response = await threadAPI.getThreads(selectedStatus);
        setThreads(response.data || []);
        const statsResponse = await threadAPI.getStats();
        setStats(statsResponse.data);
      }, 2000);
    } catch (error) {
      console.error('Error syncing mbox:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleMboxUpload = async (file: File) => {
    try {
      await threadAPI.uploadMbox(file);
      // Refresh data after upload
      setTimeout(async () => {
        const response = await threadAPI.getThreads(selectedStatus);
        setThreads(response.data || []);
        const statsResponse = await threadAPI.getStats();
        setStats(statsResponse.data);
      }, 2000);
    } catch (error) {
      console.error('Error uploading mbox:', error);
    }
  };

  const handleReset = async () => {
    if (!window.confirm('Clear all threads and messages? Next sync will re-download from PostgreSQL.org.')) return;
    try {
      await threadAPI.reset();
      setThreads([]);
      setSelectedThread(null);
      setMessages([]);
      const statsResponse = await threadAPI.getStats();
      setStats(statsResponse.data);
    } catch (error) {
      console.error('Error resetting database:', error);
    }
  };

  return (
    <div className={styles.app}>
      <header className={styles.header}>
        <div>
          <h1>PostgreSQL Mailing List Thread Analyzer</h1>
          <p>
            Identify which pgsql-hackers threads are actively being worked on
          </p>
        </div>
        <button 
          className={styles.helpButton}
          onClick={() => setIsHelpOpen(true)}
          title="View help and classification guide"
        >
          ? Help
        </button>
      </header>

      <HelpModal isOpen={isHelpOpen} onClose={() => setIsHelpOpen(false)} />

      <div className={styles.container}>
        <SyncProgressBar />
        <StatsPanel
          stats={stats}
          isLoading={isLoading}
          onMboxSync={handleMboxSync}
          onMboxUpload={handleMboxUpload}
          onReset={handleReset}
        />

        <div className={styles.mainContent}>
          <div className={styles.sidebar}>
            <FilterBar
              selectedStatus={selectedStatus}
              onStatusChange={setSelectedStatus}
            />
            <ThreadList
              threads={threads}
              onSelectThread={setSelectedThread}
              selectedStatus={selectedStatus}
            />
          </div>

          <div className={styles.detail}>
            {selectedThread ? (
              <div>
                <div className={styles.threadDetail}>
                  <h2>{selectedThread.subject}</h2>
                  <div className={styles.threadMeta}>
                    <p>
                      <strong>Author:</strong> {selectedThread.first_author}
                    </p>
                    <p>
                      <strong>Created:</strong>{' '}
                      {new Date(selectedThread.created_at).toLocaleString()}
                    </p>
                    <p>
                      <strong>Messages:</strong> {selectedThread.message_count}
                    </p>
                    <p>
                      <strong>Unique Authors:</strong>{' '}
                      {selectedThread.unique_authors}
                    </p>
                  </div>
                </div>
                <MessageThread
                  messages={messages}
                  isLoading={isMessagesLoading}
                  threadFirstAuthorEmail={selectedThread.first_author_email}
                />
              </div>
            ) : (
              <div className={styles.empty}>
                <p>Select a thread to view messages</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;
