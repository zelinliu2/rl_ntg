package match
import (
  "time"
  "errors"
  "../game"
  "../agent"
  "../channelstructs"

)


type Match struct {
  ID string
  Channels ChannelBundle

  Players []agent.Agent
  TheGame game.Game
  StartTime time.Time
}

func (m *Match) run () {
  m.matchStart()
  m.matchUnderway()
  /*

  How to determine end of match?
  * if one side resigns
  * if both sides gives up moving

  Should server actively determine end of match?
  No i guess

  */
  m.matchEnd()
}

/*
\========================================
*/

func (m *Match) matchStart(){
  m.StartTime = time.Now()
  m.TheGame = game.NewGame(agent.GetAllAgentIDs(m.Players))
  // send match start to all players
  m.broadcastStartToAllPlayers()
}

/*
\========================================
*/

func (m *Match) matchUnderway(){

  // game layer also keep track of its own version of counter
  // these are tracked by the match layer
  var moveNum uint8 = 0
  var expectedPlayer uint8 = 0

  /*
  get move from a player, then broadcast the move to all other players
  assume malicious intents from the sender though... will not process anything not from sender
  */
  for {
    select{
    case moveReceived := <- m.Channels.ChansLS2MS:
      if moveReceived.AgentID != m.Players[expectedPlayer].ID {
        log.Print("Expecting message from " + m.Players[expectedPlayer].ID + " but received a message from " + moveReceived.AgentID + " instead")
        continue
      }
      err := m.doMove(moveNum, moveReceived)
      if err != nil{
        // send a response to agent
        // or end game/match directly
      }

      // check for a win
      if m.TheGame.CheckWinCondition(){ // if a game is over, broadcast to all agents then return
        m.broadcastEndToAllPlayers()
        return
      }else { // if game continues send move message to all players
          // since these games are perfect information, we can just forward a players' move to all players
          m.broadcastMoveToAllPlayers(moveReceived.Body)
      }

      moveNum += 1
      expectedPlayer = (expectedPlayer + 1) % len(m.Players)
    default:
      m.timeoutCheck()
    }

  }


}

func (m *Match) doMove(serverMoveNum uint8, msg channelstructs.ListenerOutput) error {
  /*
  * unpack Body into a move string and a state_hash
  * send the 3 variables (player, move, state) to game
  * if no error occur, that means the move has been accepted
  * we can send this to other players
  */
  matchMoveInfo, err := ToMatchMoveInfo(msg.Body)
  if err != nil {
    return errors.New("Error parsing MatchMoveInfo(json):\n" + msg.Body + "\nDetail:"+err.Error())
  }
  if matchMoveInfo.MoveNum != serverMoveNum {
    return errors.New("Error server think move number is " + strconv.Itoa(serverMoveNum) + " but received move number is " + strconv.Itoa(matchMoveInfo.MoveNum))
  }

  // forward the move to game
  moveErr := m.TheGame.TryMove(matchMoveInfo.Move, matchMoveInfo.AfterMoveHash)
  if moveErr != nil{
    // this is different than previous errors in this section
    // previous errors would be system errors
    // therefore more critical
    // this is soft error & game specific errors like illegal moves
    // should get a response to agent ?

    // for now if we get a invalid move, we'll not progress the game
    // implement a timeout for receiving valid move
    // and use timeout to penalize the player
    return errors.New("Error move failed. Player has made an INVALID move.")
  }
  return nil
}

func (m *Match) broadcastStartToAllPlayers(){
  startInfo := MatchStartInfo{
    GamePlayers: agent.GetAllAgentIDs(m.Players),
    TimePerMove: PLAYER_TIME_PER_MOVE,
  }
  senderMessage := channelstructs.SenderMessage {
    Header: HEADER_SERVER_GAME_START,
    Body: startInfo.ToString()
  }
  senderIntake := channelstructs.SenderIntake {
    Message: senderMessage,
    AgentsToSend []m.Players,
  }
  m.Channels.ChanMS2SE <- senderIntake
}

func (m *Match) broadcastMoveToAllPlayers(body string){
  sendPackage := channelstructs.SenderIntake{
    Message: channelstruct.SenderMessage{
      Header: HEADER_SERVER_MOVE,
      Body: body,
    },
    AgentsToSend: m.Players,
  }
  m.Channels.ChanMS2SE <- sendPackage
}

func (m *Match) broadcastEndToAllPlayers(){
  sendPackage := channelstructs.SenderIntake{
    Message: channelstruct.SenderMessage{
      Header: HEADER_SERVER_GAME_END,
      Body: m.TheGame.GetMatchEndInfo().ToStinrg(),
    },
    AgentsToSend: m.Players,
  }
  m.Channels.ChanMS2SE <- sendPackage
}




/*
\========================================
*/

func (m *Match) matchEnd(){
  // send to record keeper and match
}



func (m *Match) sendMatchToRecordKeeper(){

}

/*
\========================================
*/

func (m *Match) timeoutCheck(){
  
}















// HELPR METHOD
func FindMatchByAgentID(matches []Match, agentID string)(int){
  for  i := 0; i < len(matches); i++{
    for j:= 0; j < len(matches[i].Players); j++{  // interate over matches and players
      if matches[i].Players[j].ID == agentID {
        return i
      }
    }
  }

  return -1
}
