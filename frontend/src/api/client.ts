import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

const api = axios.create({
  baseURL: `${API_BASE_URL}/api`,
  timeout: 10000,
});

export interface Thread {
  id: string;
  subject: string;
  first_message_id: string;
  first_author: string;
  first_author_email: string;
  created_at: string;
  updated_at: string;
  last_message_at: string;
  message_count: number;
  unique_authors: number;
  status: 'in-progress' | 'discussion' | 'stalled' | 'abandoned';
}

export interface Message {
  id: string;
  thread_id: string;
  message_id: string;
  subject: string;
  author: string;
  author_email: string;
  created_at: string;
}

export interface Stats {
  total_threads: number;
  total_messages: number;
  by_status: {
    [key: string]: number;
  };
  last_sync?: string;
}

export const threadAPI = {
  getThreads: (status?: string, limit?: number) =>
    api.get<Thread[]>('/threads', {
      params: { status, limit: limit || 50 },
    }),

  getThread: (id: string) =>
    api.get<Thread>(`/threads/${id}`),

  getThreadMessages: (id: string) =>
    api.get<Message[]>(`/threads/${id}/messages`),

  getStats: () =>
    api.get<Stats>('/stats'),

  sync: () =>
    api.post('/sync', {}),

  syncMbox: () =>
    api.post('/sync/mbox/all', {}),

  uploadMbox: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return api.post('/sync/mbox', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  },
};

export default api;
