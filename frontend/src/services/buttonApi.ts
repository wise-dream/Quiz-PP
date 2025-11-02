import { HardwareButton } from '../types';

// Get base API URL
const getApiUrl = () => {
  if (process.env.NODE_ENV === 'development') {
    return 'http://localhost:443';
  }
  const { protocol, host } = window.location;
  return `${protocol}//${host}`;
};

const apiUrl = getApiUrl();

// Helper function for API requests
async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${apiUrl}${endpoint}`;
  
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(errorText || `HTTP error! status: ${response.status}`);
  }

  // Handle no content responses
  if (response.status === 204) {
    return {} as T;
  }

  return response.json();
}

// Button API functions
export const buttonApi = {
  // Register a new button
  registerButton: async (macAddress: string, buttonId: string = '1', name: string = ''): Promise<HardwareButton> => {
    return apiRequest<HardwareButton>('/quiz/api/button/register', {
      method: 'POST',
      body: JSON.stringify({ macAddress, buttonId, name }),
    });
  },

  // Get all buttons
  getAllButtons: async (): Promise<HardwareButton[]> => {
    return apiRequest<HardwareButton[]>('/quiz/api/button/list');
  },

  // Get buttons by room
  getButtonsByRoom: async (roomCode: string): Promise<HardwareButton[]> => {
    return apiRequest<HardwareButton[]>(`/quiz/api/button/room/${roomCode}`);
  },

  // Get button by MAC address
  getButton: async (macAddress: string): Promise<HardwareButton> => {
    return apiRequest<HardwareButton>(`/quiz/api/button/${encodeURIComponent(macAddress)}`);
  },

  // Assign button to team
  assignButton: async (macAddress: string, roomCode: string, teamId: string): Promise<HardwareButton> => {
    return apiRequest<HardwareButton>('/quiz/api/button/assign', {
      method: 'POST',
      body: JSON.stringify({ macAddress, roomCode, teamId }),
    });
  },

  // Unassign button from team
  unassignButton: async (macAddress: string): Promise<void> => {
    await apiRequest<void>('/quiz/api/button/unassign', {
      method: 'POST',
      body: JSON.stringify({ macAddress }),
    });
  },

  // Delete button
  deleteButton: async (macAddress: string): Promise<void> => {
    await apiRequest<void>(`/quiz/api/button/${encodeURIComponent(macAddress)}`, {
      method: 'DELETE',
    });
  },

  // Press button (for hardware buttons themselves)
  pressButton: async (macAddress: string, buttonId?: string): Promise<{ success: boolean; message: string; processed: boolean }> => {
    return apiRequest<{ success: boolean; message: string; processed: boolean }>('/quiz/api/button/press', {
      method: 'POST',
      body: JSON.stringify({ macAddress, buttonId }),
    });
  },
};

