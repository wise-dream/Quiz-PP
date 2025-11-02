import React, { useState, useEffect, useCallback } from 'react';
import { buttonApi } from '../services/buttonApi';
import { HardwareButton } from '../types';
import { Plus, Trash2, Link, Unlink, RefreshCw } from 'lucide-react';
import { cn } from '../utils';

interface ButtonManagementProps {
  roomCode: string;
  teams: Record<string, { id: string; name: string; color: string }>;
}

export const ButtonManagement: React.FC<ButtonManagementProps> = ({ roomCode, teams }) => {
  const [buttons, setButtons] = useState<HardwareButton[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showAddButton, setShowAddButton] = useState(false);
  const [newButtonMac, setNewButtonMac] = useState('');
  const [newButtonName, setNewButtonName] = useState('');
  const [assigningButton, setAssigningButton] = useState<string | null>(null);

  // Helper function to convert object with numeric keys to array
  const normalizeButtonsArray = (data: any): HardwareButton[] => {
    if (!data) {
      console.log('[ButtonManagement] normalizeButtonsArray: data is null/undefined');
      return [];
    }
    if (Array.isArray(data)) {
      console.log('[ButtonManagement] normalizeButtonsArray: data is array, length:', data.length);
      return data;
    }
    if (typeof data === 'object') {
      const keys = Object.keys(data);
      console.log('[ButtonManagement] normalizeButtonsArray: data is object, keys:', keys);
      
      // Check if it's an object with numeric keys (like {"0": {...}, "1": {...}})
      if (keys.length > 0 && keys.every(k => /^\d+$/.test(k))) {
        // Convert object with numeric keys to array
        const result = keys.map(key => data[key]).filter(Boolean);
        console.log('[ButtonManagement] normalizeButtonsArray: converted numeric keys to array, length:', result.length);
        return result;
      }
      // Single object - wrap in array
      if (data.macAddress || data.id) {
        console.log('[ButtonManagement] normalizeButtonsArray: single button object found');
        return [data];
      }
      // Try to get values from object (for objects like {"button1": {...}, "button2": {...}})
      const values = Object.values(data).filter((item: any) => item && (item.macAddress || item.id));
      console.log('[ButtonManagement] normalizeButtonsArray: extracted values from object, length:', values.length);
      return values as HardwareButton[];
    }
    console.log('[ButtonManagement] normalizeButtonsArray: unknown data type:', typeof data);
    return [];
  };

  // Load all buttons (independent of room)
  const loadButtons = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      console.log('[ButtonManagement] Loading all buttons');
      const allButtonsData = await buttonApi.getAllButtons();
      console.log('[ButtonManagement] All buttons data received:', allButtonsData);
      
      // Normalize response (handle both arrays and objects with numeric keys)
      const allButtons = normalizeButtonsArray(allButtonsData);
      
      console.log('[ButtonManagement] Normalized allButtons:', allButtons.length);
      setButtons(allButtons);
    } catch (err: any) {
      setError(err.message || 'Failed to load buttons');
      console.error('[ButtonManagement] Failed to load buttons:', err);
      setButtons([]); // Set empty array on error
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadButtons();
  }, [loadButtons]);

  const handleAddButton = useCallback(async () => {
    if (!newButtonMac.trim()) {
      setError('MAC address is required');
      return;
    }

    setLoading(true);
    setError(null);
    try {
      await buttonApi.registerButton(newButtonMac, '1', newButtonName || '');
      setNewButtonMac('');
      setNewButtonName('');
      setShowAddButton(false);
      await loadButtons();
    } catch (err: any) {
      setError(err.message || 'Failed to register button');
      console.error('Failed to register button:', err);
    } finally {
      setLoading(false);
    }
  }, [newButtonMac, newButtonName, loadButtons]);

  const handleAssignButton = useCallback(async (macAddress: string, teamId: string) => {
    setAssigningButton(macAddress);
    setError(null);
    try {
      await buttonApi.assignButton(macAddress, roomCode, teamId);
      await loadButtons();
    } catch (err: any) {
      setError(err.message || 'Failed to assign button');
      console.error('Failed to assign button:', err);
    } finally {
      setAssigningButton(null);
    }
  }, [roomCode, loadButtons]);

  const handleUnassignButton = useCallback(async (macAddress: string) => {
    setError(null);
    try {
      await buttonApi.unassignButton(macAddress);
      await loadButtons();
    } catch (err: any) {
      setError(err.message || 'Failed to unassign button');
      console.error('Failed to unassign button:', err);
    }
  }, [loadButtons]);

  const handleDeleteButton = useCallback(async (macAddress: string) => {
    if (!window.confirm('Are you sure you want to delete this button?')) {
      return;
    }

    setError(null);
    try {
      await buttonApi.deleteButton(macAddress);
      await loadButtons();
    } catch (err: any) {
      setError(err.message || 'Failed to delete button');
      console.error('Failed to delete button:', err);
    }
  }, [loadButtons]);

  const formatMAC = (mac: string) => {
    // Format MAC address with colons
    const cleaned = mac.replace(/[^0-9A-F]/gi, '');
    return cleaned.match(/.{1,2}/g)?.join(':').toUpperCase() || mac.toUpperCase();
  };

  return (
    <div className="bg-white rounded-lg shadow-sm p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-gray-900">
          Физические кнопки
        </h2>
        <div className="flex items-center gap-2">
          <button
            onClick={loadButtons}
            disabled={loading}
            className="p-2 text-gray-600 hover:bg-gray-100 rounded-lg transition-colors disabled:opacity-50"
            title="Обновить список"
          >
            <RefreshCw className={cn('w-5 h-5', loading && 'animate-spin')} />
          </button>
          <button
            onClick={() => setShowAddButton(!showAddButton)}
            className="p-2 text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
            title="Добавить кнопку"
          >
            <Plus className="w-5 h-5" />
          </button>
        </div>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg">
          <p className="text-red-800 text-sm">{error}</p>
        </div>
      )}

      {showAddButton && (
        <div className="mb-4 p-4 border border-gray-200 rounded-lg bg-gray-50">
          <div className="space-y-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                MAC адрес *
              </label>
              <input
                type="text"
                value={newButtonMac}
                onChange={(e) => setNewButtonMac(e.target.value)}
                placeholder="AA:BB:CC:DD:EE:FF"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Название (опционально)
              </label>
              <input
                type="text"
                value={newButtonName}
                onChange={(e) => setNewButtonName(e.target.value)}
                placeholder="Кнопка команды 1"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <div className="flex gap-2">
              <button
                onClick={handleAddButton}
                disabled={loading || !newButtonMac.trim()}
                className="flex-1 py-2 px-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
              >
                Добавить
              </button>
              <button
                onClick={() => {
                  setShowAddButton(false);
                  setNewButtonMac('');
                  setNewButtonName('');
                }}
                className="flex-1 py-2 px-3 bg-gray-300 text-gray-700 rounded-lg hover:bg-gray-400 transition-colors"
              >
                Отмена
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="space-y-3">
        {loading && buttons.length === 0 ? (
          <div className="text-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-2"></div>
            <p className="text-gray-500">Загрузка кнопок...</p>
          </div>
        ) : buttons.length === 0 ? (
          <div className="text-center py-8 text-gray-500">
            <p>Кнопки не найдены</p>
            <p className="text-sm mt-2">Нажмите "+" чтобы добавить новую кнопку</p>
          </div>
        ) : (
          buttons.map((button) => {
            // Check if button is assigned to current room and team exists
            const isAssignedToCurrentRoom = button.roomCode === roomCode && button.teamId;
            const assignedTeam = isAssignedToCurrentRoom && button.teamId && teams[button.teamId] ? teams[button.teamId] : null;
            
            // Show team name only if assigned to current room
            const showTeamInfo = button.roomCode === roomCode && button.teamName;

            return (
              <div
                key={button.macAddress}
                className={cn(
                  'p-4 border rounded-lg',
                  isAssignedToCurrentRoom ? 'border-green-200 bg-green-50' : 'border-gray-200 bg-gray-50'
                )}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-3">
                      <h3 className="text-lg font-semibold text-gray-900">
                        {button.name || <span className="text-gray-400 italic">Без названия</span>}
                      </h3>
                      {!button.isActive && (
                        <span className="px-2 py-0.5 bg-red-100 text-red-800 text-xs rounded-full font-medium">
                          Неактивна
                        </span>
                      )}
                    </div>
                    <div className="space-y-1 mb-3">
                      <p className="text-sm text-gray-500">
                        <span className="font-medium">MAC адрес:</span>{' '}
                        <span className="font-mono text-gray-700">{formatMAC(button.macAddress)}</span>
                      </p>
                      {button.buttonId && (
                        <p className="text-xs text-gray-500">
                          <span className="font-medium">Button ID:</span> {button.buttonId}
                        </p>
                      )}
                    </div>
                    {assignedTeam ? (
                      <div className="flex items-center gap-2 mt-2">
                        <div
                          className="w-3 h-3 rounded-full"
                          style={{ backgroundColor: assignedTeam.color }}
                        />
                        <span className="text-sm text-gray-700">
                          Привязана к: <strong>{assignedTeam.name}</strong>
                        </span>
                      </div>
                    ) : showTeamInfo ? (
                      <div className="flex items-center gap-2 mt-2">
                        <span className="text-sm text-gray-700">
                          Привязана к: <strong>{button.teamName}</strong>
                        </span>
                      </div>
                    ) : button.roomCode && button.roomCode !== roomCode ? (
                      <div className="flex items-center gap-2 mt-2">
                        <span className="text-sm text-gray-500">
                          Привязана к другой комнате ({button.roomCode})
                        </span>
                      </div>
                    ) : (
                      <p className="text-sm text-gray-500 mt-2">Не привязана</p>
                    )}
                    {button.pressCount > 0 && (
                      <p className="text-xs text-gray-500 mt-1">
                        Нажатий: {button.pressCount}
                        {button.lastPress && (
                          <> • Последнее: {new Date(button.lastPress).toLocaleString('ru-RU')}</>
                        )}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center gap-2 ml-4">
                    {isAssignedToCurrentRoom ? (
                      <button
                        onClick={() => handleUnassignButton(button.macAddress)}
                        className="p-2 text-orange-600 hover:bg-orange-50 rounded-lg transition-colors"
                        title="Отвязать от команды"
                      >
                        <Unlink className="w-4 h-4" />
                      </button>
                    ) : button.roomCode && button.roomCode !== roomCode ? (
                      <div className="flex flex-col items-end gap-2">
                        <span className="text-xs text-gray-500 px-2 py-1 whitespace-nowrap">
                          Комната: {button.roomCode}
                        </span>
                        <select
                          value=""
                          onChange={(e) => {
                            if (e.target.value) {
                              handleAssignButton(button.macAddress, e.target.value);
                            }
                          }}
                          disabled={assigningButton === button.macAddress || Object.keys(teams).length === 0}
                          className="text-sm px-2 py-1 border border-blue-300 rounded-lg focus:ring-2 focus:ring-blue-500 disabled:opacity-50 bg-blue-50"
                          title="Переназначить на команду в текущей комнате"
                        >
                          <option value="">Переназначить...</option>
                          {Object.values(teams).map((team) => (
                            <option key={team.id} value={team.id}>
                              {team.name}
                            </option>
                          ))}
                        </select>
                      </div>
                    ) : (
                      <select
                        value=""
                        onChange={(e) => {
                          if (e.target.value) {
                            handleAssignButton(button.macAddress, e.target.value);
                          }
                        }}
                        disabled={assigningButton === button.macAddress || Object.keys(teams).length === 0}
                        className="text-sm px-2 py-1 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                        title="Привязать к команде"
                      >
                        <option value="">Привязать...</option>
                        {Object.values(teams).map((team) => (
                          <option key={team.id} value={team.id}>
                            {team.name}
                          </option>
                        ))}
                      </select>
                    )}
                    <button
                      onClick={() => handleDeleteButton(button.macAddress)}
                      className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                      title="Удалить кнопку"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
};

