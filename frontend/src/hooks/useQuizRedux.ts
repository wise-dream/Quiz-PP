import { useCallback, useEffect, useRef } from 'react';
import { useAppDispatch, useAppSelector } from '../store/hooks';
import { WebSocketService } from '../services/websocket';
import { Event, WebSocketMessage, EventType } from '../types';
import {
  setConnection,
  setWebSocketService,
  setRoom,
  setUser,
  setError,
  disconnect,
  setAdminData,
  setAdminPassword,
  resetQuiz,
} from '../store/quizSlice';

export const useQuiz = () => {
  console.log('🔄 [useQuiz] Redux-based hook render started');
  
  const dispatch = useAppDispatch();
  const state = useAppSelector((state) => state.quiz);
  const wsServiceRef = useRef<WebSocketService | null>(null);
  
  console.log('🔄 [useQuiz] Current Redux state:', state);

  // Load state from localStorage on mount
  useEffect(() => {
    console.log('🔄 [useQuiz] Loading state from localStorage...');
    try {
      const savedState = localStorage.getItem('quiz_state');
      if (savedState) {
        const parsedState = JSON.parse(savedState);
        console.log('🔄 [useQuiz] Loaded state from localStorage:', parsedState);
        
        if (parsedState.quiz) {
          if (parsedState.quiz.room) {
            dispatch(setRoom(parsedState.quiz.room));
          }
          if (parsedState.quiz.user) {
            dispatch(setUser(parsedState.quiz.user));
          }
          if (parsedState.quiz.adminData) {
            dispatch(setAdminData(parsedState.quiz.adminData));
          }
        }
      }
    } catch (error) {
      console.error('❌ [useQuiz] Error loading state from localStorage:', error);
    }
  }, [dispatch]);

  const connect = useCallback(async (url: string) => {
    console.log('🚀 [useQuiz] connect() called with URL:', url);
    
    // Prevent multiple connections - check if we already have a service and it's connected
    if (wsServiceRef.current) {
      const isConnected = wsServiceRef.current.isConnected();
      console.log('📊 [useQuiz] Existing service connection status:', isConnected);
      if (isConnected) {
        console.log('⚠️ [useQuiz] WebSocket already connected, skipping');
        return;
      } else {
        console.log('🔄 [useQuiz] Existing service not connected, disconnecting first');
        wsServiceRef.current.disconnect();
        wsServiceRef.current = null;
      }
    }
    
    try {
      console.log('🚀 [useQuiz] Creating WebSocket service...');
      const ws = new WebSocketService(url);
      
      console.log('🚀 [useQuiz] Attempting to connect...');
      await ws.connect();
      console.log('✅ [useQuiz] WebSocket connected successfully');
      
      console.log('🚀 [useQuiz] Setting up message handler...');
      ws.onMessage((message: WebSocketMessage) => {
        console.log('📨 [useQuiz] Message handler called with:', message);
        
        if (message.type === 'message' && message.data) {
          const event = message.data as Event;
          console.log('📦 [useQuiz] Processing event:', event);
          
          if (event.type === 'state') {
            const room = event.data as any;
            console.log('🏠 [useQuiz] Room state received:', room);
            dispatch(setRoom(room));
            
            // Save room data to localStorage for reconnection
            localStorage.setItem('quiz_room_data', JSON.stringify(room));
            console.log('💾 [useQuiz] Saved room data to localStorage:', room);
          } else if (event.type === 'error') {
            console.log('❌ [useQuiz] Error event received:', event.message);
            dispatch(setError(event.message || 'Unknown error'));
          } else if (event.type === 'room_created') {
            const room = event.data as any;
            console.log('🏠 [useQuiz] Room created:', room);
            dispatch(setRoom(room));
            
            // Set admin user data and password
            if (event.adminToken) {
              const adminUser = {
                id: `admin_${Date.now()}`,
                nickname: state.adminData?.name || 'Admin',
                role: 'admin' as const,
                roomCode: room.code
              };
              dispatch(setUser(adminUser));
              dispatch(setAdminPassword(event.adminToken));
              console.log('👤 [useQuiz] Set admin user:', adminUser);
              console.log('🔑 [useQuiz] Set admin password:', event.adminToken);
            }
            
            // Save room data to localStorage for reconnection
            localStorage.setItem('quiz_room_data', JSON.stringify(room));
            console.log('💾 [useQuiz] Saved room data to localStorage:', room);
          } else if (event.type === 'join_success') {
            const room = event.data as any;
            console.log('✅ [useQuiz] Join successful:', room);
            dispatch(setRoom(room));
            
            // Don't overwrite existing user data, just update room
            // The user data should already be set by joinRoom function
            console.log('👤 [useQuiz] Join successful, keeping existing user data');
            
            // Save room data to localStorage for reconnection
            localStorage.setItem('quiz_room_data', JSON.stringify(room));
            console.log('💾 [useQuiz] Saved room data to localStorage:', room);
          } else if (event.type === 'join_error') {
            console.log('❌ [useQuiz] Join error:', event.message);
            dispatch(setError(event.message || 'Failed to join room'));
          } else if (event.type === 'admin_reconnect_success') {
            console.log('✅ [useQuiz] Admin reconnection successful');
            const room = event.data as any;
            dispatch(setRoom(room));
            dispatch(setUser({
              id: `admin_${Date.now()}`,
              nickname: state.adminData?.name || 'Admin',
              role: 'admin',
              roomCode: room.code
            }));
          } else if (event.type === 'admin_reconnect_error') {
            console.log('❌ [useQuiz] Admin reconnection failed:', event.message);
            dispatch(setError(event.message || 'Failed to reconnect as admin'));
          } else if (event.type === 'team_created') {
            console.log('👥 [useQuiz] Team created:', event.data);
            // Room state will be updated by the server
          } else if (event.type === 'player_joined') {
            console.log('👤 [useQuiz] Player joined:', event.data);
            // Room state will be updated by the server
          } else if (event.type === 'player_left') {
            console.log('👤 [useQuiz] Player left:', event.data);
            // Room state will be updated by the server
          } else if (event.type === 'team_joined') {
            console.log('👥 [useQuiz] Player joined team:', event.data);
            // Room state will be updated by the server
          } else if (event.type === 'phase_changed') {
            console.log('🔄 [useQuiz] Phase changed:', event.phase);
            // Update room phase in Redux state
            if (state.room && event.phase) {
              const updatedRoom = { ...state.room, phase: event.phase };
              dispatch(setRoom(updatedRoom));
              console.log('🔄 [useQuiz] Updated room phase to:', event.phase);
            }
          } else if (event.type === 'start_question') {
            console.log('❓ [useQuiz] Question started');
            // Update room state with question active
            if (state.room) {
              const updatedRoom = { 
                ...state.room, 
                questionActive: true,
                firstAnswerer: '',
                questionStartTime: new Date().toISOString()
              };
              dispatch(setRoom(updatedRoom));
            }
          } else if (event.type === 'answer_received') {
            console.log('📝 [useQuiz] Answer received from:', event.userId);
            console.log('📝 [useQuiz] Answer:', event.answer);
            console.log('📝 [useQuiz] Team info:', event.teamId, event.teamName);
            console.log('📝 [useQuiz] Full event object:', JSON.stringify(event, null, 2));
            
            // Update room state with answer
            if (state.room) {
              const updatedRoom = { 
                ...state.room, 
                questionActive: false,
                firstAnswerer: event.userId || event.teamId || ''
              };
              dispatch(setRoom(updatedRoom));
              
              // If it's a hardware button press, trigger modal show via custom event
              // Check both direct properties and nested data object
              const teamId = event.teamId || (event.data as any)?.teamId;
              const teamName = event.teamName || (event.data as any)?.teamName;
              
              console.log('📝 [useQuiz] Extracted teamId:', teamId, 'teamName:', teamName);
              
              if (teamId && teamName) {
                const team = state.room.teams?.[teamId];
                const teamColor = team?.color || '#3b82f6';
                
                console.log('📝 [useQuiz] Dispatching hardwareButtonPressed event:', {
                  teamId,
                  teamName,
                  teamColor
                });
                
                // Dispatch custom event for AdminPanel to handle
                window.dispatchEvent(new CustomEvent('hardwareButtonPressed', {
                  detail: {
                    teamId: teamId,
                    teamName: teamName,
                    teamColor: teamColor
                  }
                }));
                
                console.log('✅ [useQuiz] hardwareButtonPressed event dispatched');
              } else {
                console.warn('⚠️ [useQuiz] No teamId or teamName in answer_received event');
              }
            }
          } else if (event.type === 'show_answer') {
            console.log('👁️ [useQuiz] Showing answer');
            // Answer is shown in PowerPoint, no need to update state
          } else if (event.type === 'next_question') {
            console.log('➡️ [useQuiz] Next question');
            // Reset question state
            if (state.room) {
              const updatedRoom = { 
                ...state.room, 
                questionActive: false,
                firstAnswerer: '',
                correctAnswer: '',
                questionStartTime: ''
              };
              dispatch(setRoom(updatedRoom));
            }
          } else {
            console.log('⚠️ [useQuiz] Unknown event type:', event.type);
          }
        } else if (message.type === 'error') {
          console.log('❌ [useQuiz] WebSocket error:', message.error);
          dispatch(setError(message.error || 'Connection error'));
        } else if (message.type === 'close') {
          console.log('🔌 [useQuiz] WebSocket connection closed');
          dispatch(setConnection(false));
          // Don't clear room/user data on disconnect - keep them for reconnection
        } else {
          console.log('⚠️ [useQuiz] Unknown message type:', message.type);
        }
      });

      console.log('🚀 [useQuiz] Setting WebSocket service...');
      wsServiceRef.current = ws;
      dispatch(setWebSocketService(ws));
      dispatch(setConnection(true));
      
      // Check for admin reconnection after a short delay to allow state to update
      setTimeout(() => {
        // Get admin data from localStorage for reconnection
        const savedAdminData = localStorage.getItem('quiz_admin_data');
        const savedRoomData = localStorage.getItem('quiz_room_data');
        
        console.log('🔄 [useQuiz] Checking for admin reconnection...');
        console.log('🔄 [useQuiz] Saved admin data:', savedAdminData);
        console.log('🔄 [useQuiz] Saved room data:', savedRoomData);
        
        if (savedRoomData && savedAdminData) {
          try {
            const adminData = JSON.parse(savedAdminData);
            const roomData = JSON.parse(savedRoomData);
            
            console.log('🔄 [useQuiz] Attempting admin reconnection...');
            console.log('🔄 [useQuiz] Room:', roomData.code);
            console.log('🔄 [useQuiz] Admin:', adminData.name);
            
            // Send admin reconnection event
            const reconnectEvent = {
              type: 'admin_reconnect' as const,
              roomCode: roomData.code,
              adminName: adminData.name,
              adminEmail: adminData.email,
            };
            
            console.log('🔄 [useQuiz] Sending admin reconnect event:', reconnectEvent);
            ws.send(reconnectEvent);
          } catch (error) {
            console.error('❌ [useQuiz] Error parsing saved data:', error);
          }
        } else {
          console.log('🔄 [useQuiz] No admin reconnection needed');
        }
      }, 2000); // Wait for state to update
      
      console.log('✅ [useQuiz] Connection setup complete');
    } catch (error) {
      console.error('❌ [useQuiz] Failed to connect to WebSocket:', error);
      dispatch(setError('Failed to connect to server'));
    }
  }, [dispatch]);

  const disconnectWebSocket = useCallback(() => {
    console.log('🔌 [useQuiz] disconnect() called');
    if (wsServiceRef.current) {
      console.log('🔌 [useQuiz] Disconnecting WebSocket service...');
      wsServiceRef.current.disconnect();
      wsServiceRef.current = null;
      dispatch(disconnect());
      console.log('✅ [useQuiz] Disconnected successfully');
    } else {
      console.log('⚠️ [useQuiz] No WebSocket service to disconnect');
    }
  }, [dispatch]);

  const sendEvent = useCallback((event: Event) => {
    console.log('📤 [useQuiz] sendEvent() called');
    console.log('📤 [useQuiz] Event to send:', event);
    
    const currentWsService = wsServiceRef.current || state.wsService;
    console.log('📤 [useQuiz] Using WebSocket service:', currentWsService);
    
    if (!currentWsService) {
      console.error('❌ [useQuiz] WebSocket service not available');
      dispatch(setError('WebSocket сервис недоступен. Перезагрузите страницу.'));
      return;
    }
    
    if (!currentWsService.isConnected()) {
      console.error('❌ [useQuiz] WebSocket is not connected');
      dispatch(setError('WebSocket не подключен. Проверьте подключение к серверу.'));
      return;
    }
    
    try {
      currentWsService.send(event);
      console.log('✅ [useQuiz] Event sent successfully:', event);
    } catch (error) {
      console.error('❌ [useQuiz] Error sending event:', error);
      dispatch(setError('Ошибка отправки сообщения: ' + error));
    }
  }, [dispatch, state.wsService]);

  const createRoom = useCallback((adminName?: string, adminEmail?: string) => {
    console.log('🏠 [useQuiz] createRoom() called');
    console.log('🏠 [useQuiz] Admin name:', adminName);
    console.log('🏠 [useQuiz] Admin email:', adminEmail);
    
    // Store admin data if provided
    if (adminName) {
      const adminData = {
        name: adminName,
        email: adminEmail || '',
      };
      dispatch(setAdminData(adminData));
      
      // Save to localStorage for reconnection
      localStorage.setItem('quiz_admin_data', JSON.stringify(adminData));
      console.log('💾 [useQuiz] Saved admin data to localStorage:', adminData);
    }
    
    const event = { type: 'create_room' as const };
    console.log('🏠 [useQuiz] Calling sendEvent with:', event);
    sendEvent(event);
  }, [sendEvent, dispatch, state.adminData?.name]);

  const joinRoom = useCallback((roomCode: string, nickname: string) => {
    console.log('🚪 [useQuiz] joinRoom() called');
    console.log('🚪 [useQuiz] Room code:', roomCode);
    console.log('🚪 [useQuiz] Nickname:', nickname);
    
    const userId = `user_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    console.log('🚪 [useQuiz] Generated user ID:', userId);
    
    console.log('🚪 [useQuiz] Updating user state...');
    dispatch(setUser({
      id: userId,
      nickname,
      role: 'participant',
      roomCode,
    }));

    const event = {
      type: 'join' as const,
      quizId: roomCode,
      userId,
      nickname,
    };
    console.log('🚪 [useQuiz] Calling sendEvent with:', event);
    sendEvent(event);
  }, [sendEvent, dispatch]);

  const authenticateAdmin = useCallback((roomCode: string, password: string) => {
    console.log('🔐 [useQuiz] authenticateAdmin() called');
    console.log('🔐 [useQuiz] Room code:', roomCode);
    console.log('🔐 [useQuiz] Password:', password ? '[HIDDEN]' : '[EMPTY]');
    
    const userId = `admin_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    console.log('🔐 [useQuiz] Generated admin user ID:', userId);
    
    dispatch(setUser({
      id: userId,
      nickname: 'Admin',
      role: 'admin',
      roomCode,
    }));

    const event = {
      type: 'admin_auth' as const,
      roomCode,
      password,
    };
    console.log('🔐 [useQuiz] Calling sendEvent with:', event);
    sendEvent(event);
  }, [sendEvent, dispatch]);

  const joinTeam = useCallback((teamId: string) => {
    if (!state.user) return;

    sendEvent({
      type: 'join_team',
      userId: state.user.id,
      teamId,
      nickname: state.user.nickname,
    });
  }, [sendEvent, state.user]);

  const createTeam = useCallback((teamName: string, teamColor: string) => {
    if (!state.room) {
      console.error('No room available for createTeam');
      return;
    }
    
    sendEvent({
      type: 'create_team',
      quizId: state.room.code,
      teamName,
      teamColor,
    });
  }, [sendEvent, state.room]);

  const setGamePhase = useCallback((phase: string, delayMs?: number) => {
    if (!state.room) {
      console.error('No room available for setGamePhase');
      return;
    }
    
    sendEvent({
      type: 'host_set_state',
      quizId: state.room.code,
      phase: phase as any,
      delayMs,
    });
  }, [sendEvent, state.room]);

  const sendClick = useCallback((buttonId: string) => {
    if (!state.user) return;

    sendEvent({
      type: 'click',
      userId: state.user.id,
      buttonId,
      tsClient: Date.now(),
    });
  }, [sendEvent, state.user]);

  const leaveRoom = useCallback(() => {
    console.log('🚪 [useQuiz] leaveRoom() called');
    
    if (!state.user) {
      console.log('⚠️ [useQuiz] No user to leave room');
      return;
    }

    console.log('🚪 [useQuiz] Sending leave event for user:', state.user.id);
    
    // Send leave event to server
    sendEvent({
      type: 'leave',
      userId: state.user.id,
      roomCode: state.user.roomCode,
    });

    // Clear local state and localStorage
    dispatch(resetQuiz());
    
    console.log('✅ [useQuiz] Left room successfully');
  }, [sendEvent, state.user, dispatch]);

  const startQuestion = useCallback(() => {
    console.log('❓ [useQuiz] startQuestion() called');
    
    sendEvent({
      type: 'start_question',
    });
  }, [sendEvent]);

  const sendAnswer = useCallback((answer: string) => {
    console.log('📝 [useQuiz] sendAnswer() called');
    console.log('📝 [useQuiz] Answer:', answer);
    
    if (!state.user) {
      console.log('⚠️ [useQuiz] No user to send answer');
      return;
    }
    
    sendEvent({
      type: 'answer_received',
      userId: state.user.id,
      answer,
    });
  }, [sendEvent, state.user]);

  const showAnswer = useCallback(() => {
    console.log('👁️ [useQuiz] showAnswer() called');
    
    sendEvent({
      type: 'show_answer',
    });
  }, [sendEvent]);

  const nextQuestion = useCallback(() => {
    console.log('➡️ [useQuiz] nextQuestion() called');
    
    sendEvent({
      type: 'next_question',
    });
  }, [sendEvent]);

  useEffect(() => {
    return () => {
      disconnectWebSocket();
    };
  }, [disconnectWebSocket]);

  return {
    ...state,
    connect,
    disconnect: disconnectWebSocket,
    createRoom,
    joinRoom,
    authenticateAdmin,
    joinTeam,
    createTeam,
    setGamePhase,
    sendClick,
    leaveRoom,
    startQuestion,
    sendAnswer,
    showAnswer,
    nextQuestion,
  };
};
