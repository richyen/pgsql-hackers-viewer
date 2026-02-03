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
  const [allThreads, setAllThreads] = useState<Thread[]>([]); // Store all threads for client-side filtering
  const [selectedThread, setSelectedThread] = useState<Thread | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [selectedStatus, setSelectedStatus] = useState<string | undefined>();
  const [searchTerm, setSearchTerm] = useState<string>('');
  const [isLoading, setIsLoading] = useState(false);
  const [isMessagesLoading, setIsMessagesLoading] = useState(false);
  const [isHelpOpen, setIsHelpOpen] = useState(false);
  const [highlightedMessageId, setHighlightedMessageId] = useState<string | null>(null);

  // Handle URL parameters for deep-linking
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const threadId = params.get('thread');
    const messageId = params.get('message');

    console.log('Deep link params:', { threadId, messageId });

    if (threadId) {
      // Load the specific thread
      threadAPI.getThread(threadId).then(response => {
        console.log('Loaded thread from URL:', response.data);
        setSelectedThread(response.data);
        if (messageId) {
          setHighlightedMessageId(messageId);
        }
      }).catch(error => {
        console.error('Error loading thread from URL:', error);
        alert('Thread not found. The link may be invalid or the thread may have been deleted.');
      });
    }
  }, []);

  // Update URL when thread selection changes (but not during initial load from URL)
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const urlThreadId = params.get('thread');
    
    if (selectedThread && selectedThread.id !== urlThreadId) {
      // Update URL without full page reload
      const newUrl = `${window.location.pathname}?thread=${selectedThread.id}`;
      window.history.pushState({}, '', newUrl);
      setHighlightedMessageId(null); // Clear highlight when manually selecting a thread
    } else if (!selectedThread && urlThreadId) {
      // Clear URL params when thread is deselected
      window.history.pushState({}, '', window.location.pathname);
    }
  }, [selectedThread]);

  // Handle browser back/forward navigation
  useEffect(() => {
    const handlePopState = () => {
      const params = new URLSearchParams(window.location.search);
      const threadId = params.get('thread');
      const messageId = params.get('message');

      if (threadId) {
        threadAPI.getThread(threadId).then(response => {
          setSelectedThread(response.data);
          if (messageId) {
            setHighlightedMessageId(messageId);
          }
        }).catch(error => {
          console.error('Error loading thread from history:', error);
        });
      } else {
        setSelectedThread(null);
        setHighlightedMessageId(null);
      }
    };

    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

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
        setAllThreads(response.data || []);
        // Don't clear selectedThread if we're loading from URL params
        const params = new URLSearchParams(window.location.search);
        const threadIdFromUrl = params.get('thread');
        if (!threadIdFromUrl) {
          setSelectedThread(null);
          setMessages([]);
        }
      } catch (error) {
        console.error('Error fetching threads:', error);
      } finally {
        setIsLoading(false);
      }
    };

    fetchThreads();
  }, [selectedStatus]);

  // Filter threads by search term
  useEffect(() => {
    if (!searchTerm.trim()) {
      setThreads(allThreads);
    } else {
      const filtered = allThreads.filter(thread =>
        thread.subject.toLowerCase().includes(searchTerm.toLowerCase())
      );
      setThreads(filtered);
    }
  }, [searchTerm, allThreads]);

  // Fetch messages when thread is selected
  useEffect(() => {
    console.log('Thread selection changed:', selectedThread?.id);
    if (!selectedThread) {
      setMessages([]);
      return;
    }

    const fetchMessages = async () => {
      setIsMessagesLoading(true);
      try {
        console.log('Fetching messages for thread:', selectedThread.id);
        const response = await threadAPI.getThreadMessages(selectedThread.id);
        console.log('Received messages:', response.data?.length || 0);
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
        setAllThreads(response.data || []);
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
        setAllThreads(response.data || []);
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
      setAllThreads([]);
      setSelectedThread(null);
      setMessages([]);
      const statsResponse = await threadAPI.getStats();
      setStats(statsResponse.data);
    } catch (error) {
      console.error('Error resetting database:', error);
    }
  };

  console.log('App render state:', { 
    selectedThread: selectedThread?.id, 
    messagesCount: messages.length, 
    isMessagesLoading,
    highlightedMessageId 
  });

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
              searchTerm={searchTerm}
              onSearchChange={setSearchTerm}
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
                  highlightedMessageId={highlightedMessageId}
                  threadId={selectedThread.id}
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
