package server

import (
	"encoding/json"
	"github.com/brianvoe/gofakeit"
	"github.com/gofrs/uuid"
	"github.com/olahol/melody"
	serverv1 "github.com/petomalina/thatoneroom/server/pkg/api/thatoneroom/server/v1"
	"go.uber.org/zap"
	"math/rand"
	"sort"
	"sync"
)

type WebSocketService struct {
	M        *melody.Melody
	game     *Game
	mx       sync.RWMutex
	sessions map[string]*Session
}

type Session struct {
	S     *melody.Session
	ID    string
	Token string
}

func NewWSService() *WebSocketService {
	svc := &WebSocketService{
		M:        melody.New(),
		game:     NewGame(),
		sessions: make(map[string]*Session),
	}
	svc.game.Start()
	svc.watchChanges()
	svc.M.HandleConnect(svc.Connect)
	svc.M.HandleDisconnect(svc.Disconnect)
	svc.M.HandleMessage(svc.Message)
	return svc
}

func (s *WebSocketService) Connect(m *melody.Session) {

}

func (s *WebSocketService) Disconnect(m *melody.Session) {
	sessionAny, exists := m.Get("session")
	if !exists {
		return
	}
	session := sessionAny.(*Session)
	s.game.ActionChannel <- &RemovePlayerAction{
		ID: session.ID,
	}
}

func (s *WebSocketService) Message(m *melody.Session, msg []byte) {
	var sMsg serverv1.Message
	if err := json.Unmarshal(msg, &sMsg); err != nil {
		sendMsg(m, err)
		return
	}
	sessionAny, exists := m.Get("session")
	if !exists {
		sessionAny = &Session{
			S: m,
		}
	}
	session := sessionAny.(*Session)
	switch sMsg.Type {
	case serverv1.TypePlayerAuthenticate:
		var data serverv1.PlayerAuthenticate
		json.Unmarshal(sMsg.Data, &data)
		s.HandleAuthenticate(session, data)
	case serverv1.TypePlayerConnect:
		var data serverv1.PlayerConnect
		json.Unmarshal(sMsg.Data, &data)
		s.HandleConnect(session, data)
	case serverv1.TypePlayerMove:
		var data serverv1.PlayerMove
		json.Unmarshal(sMsg.Data, &data)
		s.HandlePlayerMove(session, data)
	case serverv1.TypePlayerSpawnObject:
		var data serverv1.PlayerSpawnObject
		json.Unmarshal(sMsg.Data, &data)
		s.HandleSpawnObject(session, data)
	case serverv1.TypePlayerPickupItem:
		var data serverv1.PlayerPickupItem
		json.Unmarshal(sMsg.Data, &data)
		s.HandlePickupItem(session, data)
	case serverv1.TypePlayerDead:
		var data serverv1.PlayerDead
		json.Unmarshal(sMsg.Data, &data)
		s.HandlePlayerDead(session, data)
	default:
		sendMsg(m, "unknown")
	}
}

func (s *WebSocketService) watchChanges() {
	go func() {
		for {
			change := <-s.game.ChangeChannel
			var msg serverv1.Message
			switch val := change.(type) {
			case NewPlayerChange:
				msg = serverv1.NewServerAddPlayer(serverv1.ServerAddPlayer{
					ID: val.Object.ID,
					X:  val.Object.Coords.X,
					Y:  val.Object.Coords.Y,
				})
			case PickupItemChange:
				msg = serverv1.NewServerPickupItem(val.PlayerID, val.ItemID, val.Type, val.Item)
			case MoveChange:
				msg = serverv1.NewServerMove(val.Object.ID, val.Object.Coords.X, val.Object.Coords.Y, val.Object.State)
			case RemovePlayerChange:
				msg = serverv1.NewServerRemovePlayer(val.ID)
			case SpawnObjectChange:
				msg = serverv1.NewServerSpawnObject(serverv1.ServerSpawnObject{
					ID: val.ID,
					PlayerSpawnObject: serverv1.PlayerSpawnObject{
						Item:      val.Item,
						PlayerID:  val.PlayerID,
						Type:      val.Type,
						X:         val.X,
						Y:         val.Y,
						VelocityX: val.VelX,
						VelocityY: val.VelY,
					},
				})
			case ResetChange:
				msg = serverv1.NewServerState(s.getState())
			case PlayerDeadChange:
				msg = serverv1.NewServerPlayerDead(serverv1.ServerPlayerDead{
					ID:       val.PlayerID,
					KilledBy: val.KilledBy,
				})
			default:
				zap.L().Warn("unhandled msg", zap.Any("val", val))
				continue
			}

			b, _ := json.Marshal(msg)
			zap.L().Info("broadcast", zap.Any("msg", msg))
			if err := s.M.Broadcast(b); err != nil {
				zap.L().Error("broadcast", zap.Error(err))

			}
		}
	}()
}

func (s *WebSocketService) HandleAuthenticate(ps *Session, data serverv1.PlayerAuthenticate) {
	zap.L().Info("handle", zap.Any("data", data))

	if data.Token != "" {
		if ps.Token != data.Token {
			sendMsg(ps.S, serverv1.NewServerAuthenticate(serverv1.ServerAuthenticate{}))
			return
		}
		sendMsg(ps.S, serverv1.NewServerAuthenticate(serverv1.ServerAuthenticate{
			Success: true,
			Token:   data.Token,
			ID:      ps.ID,
			Name:    "@TODO",
		}))
		sendMsg(ps.S, serverv1.NewServerState(s.getState()))

		return
	}
	id := uuid.Must(uuid.NewV4()).String()
	token := uuid.Must(uuid.NewV4()).String()
	session := &Session{
		ID:    id,
		Token: token,
		S:     ps.S,
	}
	ps.S.Set("session", session)

	sendMsg(ps.S, serverv1.NewServerAuthenticate(serverv1.ServerAuthenticate{
		Success: true,
		Token:   session.Token,
		ID:      session.ID,
		Name:    gofakeit.HackerNoun(),
	}))
	sendMsg(ps.S, serverv1.NewServerState(s.getState()))

	s.game.ActionChannel <- &AuthPlayerAction{
		ID: ps.ID,
	}
}

func (s *WebSocketService) HandleConnect(ps *Session, data serverv1.PlayerConnect) {
	zap.L().Info("handle", zap.Any("data", data))
	sendMsg(ps.S, serverv1.NewServerState(s.getState()))
	if data.Name == "" {
		data.Name = gofakeit.HackerNoun()
	}
	s.game.ActionChannel <- &AddPlayerAction{
		ID:    ps.ID,
		Name:  data.Name,
		X:     data.X,
		Y:     data.Y,
		Color: rand.Intn(0xffffff),
	}
}

func (s *WebSocketService) HandlePlayerMove(ps *Session, data serverv1.PlayerMove) {
	//zap.L().Info("handle", zap.Any("data", data))

	if ps.ID == "" {
		sendMsg(ps.S, "authorize first")
		return
	}
	s.game.ActionChannel <- &MoveAction{
		ID: ps.ID,
		To: Coords{
			X: data.X,
			Y: data.Y,
		},
		State: data.State,
	}
}

func (s *WebSocketService) HandleSpawnObject(ps *Session, data serverv1.PlayerSpawnObject) {
	zap.L().Info("handle", zap.Any("data", data))

	if ps.ID == "" {
		sendMsg(ps.S, "authorize first")
		return
	}

	s.game.ActionChannel <- &SpawnObjectAction{
		ID:       shortID(),
		PlayerID: data.PlayerID,
		Type:     data.Type,
		X:        data.X,
		Y:        data.Y,
		VelX:     data.VelocityX,
		VelY:     data.VelocityY,
	}

}
func (s *WebSocketService) HandlePickupItem(ps *Session, data serverv1.PlayerPickupItem) {
	zap.L().Info("handle", zap.Any("data", data))

	if ps.ID == "" {
		sendMsg(ps.S, "authorize first")
		return
	}

	s.game.ActionChannel <- &PickupItemAction{
		PlayerID: ps.ID,
		ItemID:   data.ID,
		Type:     data.Type,
		Item:     data.Item,
	}
}

func (s *WebSocketService) HandlePlayerDead(ps *Session, data serverv1.PlayerDead) {
	zap.L().Info("handle", zap.Any("data", data))

	if ps.ID == "" {
		sendMsg(ps.S, "authorize first")
		return
	}

	s.game.ActionChannel <- &PlayerDeadAction{
		PlayerID: ps.ID,
		KilledBy: data.KilledBy,
	}

}

func (s *WebSocketService) getState() serverv1.ServerState {
	objs := s.game.Objects()

	state := serverv1.ServerState{
		EndAt:   s.game.startAt.Add(roundDuration),
		Objects: make([]serverv1.Object, 0, len(objs)),
	}
	for _, v := range objs {
		items := make([]serverv1.Item, 0, len(v.Inventory))
		for _, item := range v.Inventory {
			items = append(items, serverv1.Item{
				Type:  item.Type,
				Count: item.Count,
			})
		}
		state.Objects = append(state.Objects, serverv1.Object{
			ID:        v.ID,
			Name:      v.Name,
			Type:      v.Type,
			Item:      v.Item,
			X:         v.Coords.X,
			Y:         v.Coords.Y,
			Inventory: items,
		})
		if v.Type == ObjectPlayer {
			state.Leaderboard = append(state.Leaderboard, serverv1.PlayerScore{
				ID:    v.ID,
				Name:  v.Name,
				Score: v.Score,
			})
		}
	}
	sort.Slice(state.Leaderboard, func(i, j int) bool {
		return state.Leaderboard[i].Score > state.Leaderboard[j].Score
	})
	return state
}
func sendMsg(m *melody.Session, msg any) {
	b, err := json.Marshal(msg)
	if err != nil {
		m.CloseWithMsg([]byte(err.Error()))
		return
	}
	m.Write(b)
}
