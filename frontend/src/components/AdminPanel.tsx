import React, { useState, useCallback, useMemo, useEffect } from 'react';
import { useQuiz } from '../hooks/useQuizRedux';
import { teamColors, getTeamColor, cn } from '../utils';
import { Plus, Users, Play, Square, Settings, LogOut, X } from 'lucide-react';
import { ButtonManagement } from './ButtonManagement';

export const AdminPanel: React.FC = () => {
  const { room, user, createTeam, setGamePhase, error, isConnected, leaveRoom } = useQuiz();
  const [showCreateTeam, setShowCreateTeam] = useState(false);
  const [newTeamName, setNewTeamName] = useState('');
  const [newTeamColor, setNewTeamColor] = useState(teamColors[0].value);
  const [showAnswerModal, setShowAnswerModal] = useState(false);
  const [answerTeam, setAnswerTeam] = useState<{ name: string; color: string; teamId: string } | null>(null);

  // Listen for hardware button press events
  useEffect(() => {
    const handleHardwareButton = (event: CustomEvent) => {
      const { teamId, teamName, teamColor } = event.detail;
      setAnswerTeam({ name: teamName, color: teamColor, teamId });
      setShowAnswerModal(true);
    };

    window.addEventListener('hardwareButtonPressed', handleHardwareButton as EventListener);
    return () => {
      window.removeEventListener('hardwareButtonPressed', handleHardwareButton as EventListener);
    };
  }, []);

  const closeAnswerModal = useCallback(() => {
    setShowAnswerModal(false);
    setAnswerTeam(null);
  }, []);

  // All hooks must be called before any conditional returns
  const handleCreateTeam = useCallback(() => {
    if (!newTeamName.trim()) return;
    
    createTeam(newTeamName, newTeamColor);
    setNewTeamName('');
    setShowCreateTeam(false);
  }, [newTeamName, newTeamColor, createTeam]);

  const toggleCreateTeam = useCallback(() => {
    setShowCreateTeam(prev => !prev);
  }, []);

  const handleTeamNameChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setNewTeamName(e.target.value);
  }, []);

  const handleTeamColorChange = useCallback((color: string) => {
    setNewTeamColor(color);
  }, []);

  // Memoize team list to prevent unnecessary re-renders
  const teamList = useMemo(() => {
    if (!room?.teams) return [];
    return Object.values(room.teams).map((team) => {
      const color = getTeamColor(team.color);
      return (
        <div
          key={team.id}
          className="p-3 border border-gray-200 rounded-lg"
        >
          <div className="flex items-center gap-3">
            <div
              className="w-4 h-4 rounded-full"
              style={{ backgroundColor: team.color }}
            />
            <div className="flex-1">
              <h3 className="font-medium text-gray-900">{team.name}</h3>
              <p className="text-sm text-gray-500">
                {team.players.length} участников
              </p>
            </div>
            <div className="text-right">
              <p className="text-lg font-bold text-gray-900">{team.score}</p>
              <p className="text-xs text-gray-500">очков</p>
            </div>
          </div>
        </div>
      );
    });
  }, [room?.teams]);

  // Memoize player list
  const playerList = useMemo(() => {
    if (!room?.players || !room?.teams) return [];
    return Object.values(room.players).map((player) => {
      const playerTeam = Object.values(room.teams).find(team => 
        team.players.includes(player.userId)
      );
      const teamColor = playerTeam ? getTeamColor(playerTeam.color) : null;
      
      return (
        <div
          key={player.userId}
          className="flex items-center gap-3 p-3 bg-gray-50 rounded-lg"
        >
          <div className="flex-1">
            <p className="font-medium text-gray-900">{player.name}</p>
            {playerTeam && (
              <p className="text-sm text-gray-500">{playerTeam.name}</p>
            )}
          </div>
          <div className="text-right">
            <p className="text-sm font-medium text-gray-900">{player.clickCount}</p>
            <p className="text-xs text-gray-500">кликов</p>
          </div>
          {teamColor && (
            <div
              className="w-3 h-3 rounded-full"
              style={{ backgroundColor: teamColor.value }}
            />
          )}
        </div>
      );
    });
  }, [room?.players, room?.teams]);

  console.log('🔄 [AdminPanel] AdminPanel render started');
  console.log('🔄 [AdminPanel] Current room:', room);
  console.log('🔄 [AdminPanel] Current user:', user);
  console.log('🔄 [AdminPanel] Current error:', error);

  if (!room || !user) {
    console.log('❌ [AdminPanel] Missing room or user data, returning null');
    console.log('❌ [AdminPanel] Room:', room);
    console.log('❌ [AdminPanel] User:', user);
    return (
      <div className="min-h-screen bg-gray-100 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Загрузка админ-панели...</p>
          <p className="text-sm text-gray-500 mt-2">
            {!room && 'Ожидание данных комнаты...'}
            {!user && 'Ожидание данных пользователя...'}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 p-4">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <div className="bg-white rounded-lg shadow-sm p-6 mb-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">
                Админ-панель
              </h1>
              <p className="text-gray-600">
                Комната: <span className="font-mono font-bold text-blue-600">{room.code}</span>
              </p>
              <div className="flex items-center gap-2 mt-1">
                <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`}></div>
                <span className="text-xs text-gray-500">
                  {isConnected ? 'Подключено' : 'Отключено'}
                </span>
              </div>
            </div>
            <div className="flex items-center gap-4">
              <div className="text-right">
                <p className="text-sm text-gray-500">Участников</p>
                <p className="text-2xl font-bold text-green-600">
                  {Object.keys(room.players).length}
                </p>
              </div>
              <button
                onClick={leaveRoom}
                className="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2"
                title="Выйти из комнаты"
              >
                <LogOut className="w-4 h-4" />
                Выйти
              </button>
            </div>
          </div>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
            <p className="text-red-800 text-sm">{error}</p>
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Game Controls */}
          <div className="bg-white rounded-lg shadow-sm p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
              <Settings className="w-5 h-5" />
              Управление игрой
            </h2>
            <div className="space-y-3">
              {/* Кнопка "Начало игры" - показывается только когда игра не запущена */}
              {(room.phase === 'lobby' || room.phase === 'finished') && (
                <button
                  onClick={() => setGamePhase('started')}
                  className={cn(
                    'flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-colors w-full',
                    'bg-blue-600 hover:bg-blue-700 text-white',
                    'focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500'
                  )}
                >
                  <Play className="w-4 h-4" />
                  Начало игры
                </button>
              )}

              {/* Кнопка "Активировать кнопку" / "Деактивировать кнопку" - показывается только когда игра начата */}
              {(room.phase === 'started' || room.phase === 'active') && (
                <button
                  onClick={() => {
                    if (room.phase === 'started') {
                      setGamePhase('active');
                    } else if (room.phase === 'active') {
                      setGamePhase('started');
                    }
                  }}
                  className={cn(
                    'flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-colors w-full',
                    'focus:outline-none focus:ring-2 focus:ring-offset-2',
                    room.phase === 'active'
                      ? 'bg-yellow-600 hover:bg-yellow-700 text-white focus:ring-yellow-500'
                      : 'bg-green-600 hover:bg-green-700 text-white focus:ring-green-500'
                  )}
                >
                  <Play className="w-4 h-4" />
                  {room.phase === 'active' ? 'Деактивировать кнопку' : 'Активировать кнопку'}
                </button>
              )}

              {/* Кнопка "Завершение" - показывается только когда игра начата */}
              {(room.phase === 'started' || room.phase === 'active') && (
                <button
                  onClick={() => setGamePhase('finished')}
                  className={cn(
                    'flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-colors w-full',
                    'bg-red-600 hover:bg-red-700 text-white',
                    'focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500'
                  )}
                >
                  <Square className="w-4 h-4" />
                  Завершение
                </button>
              )}

              {/* Кнопка "Ожидание" - показывается только когда не в лобби (чтобы вернуться обратно) */}
              {room.phase !== 'lobby' && (
                <button
                  onClick={() => setGamePhase('lobby')}
                  className={cn(
                    'flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-colors w-full',
                    'bg-gray-600 hover:bg-gray-700 text-white',
                    'focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-500'
                  )}
                >
                  <Users className="w-4 h-4" />
                  Ожидание
                </button>
              )}
            </div>
            <div className="mt-4 p-3 bg-blue-50 rounded-lg">
              <p className="text-sm text-blue-800">
                Текущая фаза: <span className="font-semibold">{room.phase}</span>
              </p>
            </div>
          </div>

          {/* Teams Management */}
          <div className="bg-white rounded-lg shadow-sm p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
                <Users className="w-5 h-5" />
                Команды
              </h2>
              <button
                onClick={toggleCreateTeam}
                className="p-2 text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
              >
                <Plus className="w-5 h-5" />
              </button>
            </div>

            {showCreateTeam && (
              <div className="mb-4 p-4 border border-gray-200 rounded-lg">
                <div className="space-y-3">
                  <input
                    type="text"
                    value={newTeamName}
                    onChange={handleTeamNameChange}
                    placeholder="Название команды"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                  <div className="grid grid-cols-4 gap-2">
                    {teamColors.map((color) => (
                      <button
                        key={color.value}
                        onClick={() => handleTeamColorChange(color.value)}
                        className={cn(
                          'w-8 h-8 rounded-full border-2 transition-all',
                          newTeamColor === color.value ? 'border-gray-800' : 'border-gray-300'
                        )}
                        style={{ backgroundColor: color.value }}
                      />
                    ))}
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={handleCreateTeam}
                      className="flex-1 py-2 px-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                    >
                      Создать
                    </button>
                    <button
                      onClick={toggleCreateTeam}
                      className="flex-1 py-2 px-3 bg-gray-300 text-gray-700 rounded-lg hover:bg-gray-400 transition-colors"
                    >
                      Отмена
                    </button>
                  </div>
                </div>
              </div>
            )}

            <div className="space-y-3">
              {teamList}
            </div>
          </div>

          {/* Participants */}
          <div className="bg-white rounded-lg shadow-sm p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
              <Users className="w-5 h-5" />
              Участники
            </h2>
            <div className="space-y-2">
              {playerList}
            </div>
          </div>
        </div>

        {/* Button Management - Full width */}
        <div className="mt-6">
          <ButtonManagement 
            roomCode={room.code} 
            teams={room.teams || {}}
          />
        </div>
      </div>

      {/* Answer Modal for Hardware Button */}
      {showAnswerModal && answerTeam && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-xl font-bold text-gray-900">
                Нажата физическая кнопка!
              </h3>
              <button
                onClick={closeAnswerModal}
                className="text-gray-400 hover:text-gray-600 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            
            <div className="mb-6">
              <p className="text-gray-600 mb-4">
                Команда ответила через физическую кнопку:
              </p>
              <div
                className="p-6 rounded-lg text-white text-center font-bold text-2xl"
                style={{ backgroundColor: answerTeam.color }}
              >
                {answerTeam.name}
              </div>
            </div>

            <div className="flex gap-3">
              <button
                onClick={closeAnswerModal}
                className="flex-1 py-2 px-4 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
              >
                Понятно
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
