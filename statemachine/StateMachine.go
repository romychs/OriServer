//
// Машина состояний
//
package statemachine

import (
	log "github.com/sirupsen/logrus"
	"strconv"
)

const (
	S_START = iota
	S_READY
	S_LINKS_LOAD
	S_PING
	S_ORI_CMD
	S_ORI_CMD_SEND_VER
	S_ORI_CMD_SEND_DATE
	S_ORI_CMD_SEND_TIME
	S_ORI_CMD_CLIENT_REG
	S_ORI_CMD_FILE
	S_ORI_CMD_CHK_SERVER_REG

	S_ORI_FILE
	S_ORI_FILE_DIR

	// дальше остальные константы статуса
	S_SEND_ERROR = 0xfffe
	S_FAIL       = 0xffff
)

var stateNames = map[uint]string{S_START: "S_START", S_READY: "S_READY", S_LINKS_LOAD: "S_LINKS_LOAD", S_PING: "S_PING",
	S_ORI_CMD: "S_ORI_CMD", S_ORI_CMD_SEND_VER: "S_ORI_CMD_SEND_VER", S_ORI_CMD_SEND_DATE: "S_ORI_CMD_SEND_DATE",
	S_ORI_CMD_SEND_TIME: "S_ORI_CMD_SEND_TIME", S_ORI_CMD_CLIENT_REG: "S_ORI_CMD_CLIENT_REG",
	S_ORI_CMD_CHK_SERVER_REG: "S_ORI_CMD_CHK_SERVER_REG", S_ORI_CMD_FILE: "S_ORI_CMD_FILE", S_ORI_FILE: "S_ORI_FILE",
	S_ORI_FILE_DIR: "S_ORI_FILE_DIR", S_SEND_ERROR: "S_SEND_ERROR", S_FAIL: "S_FAIL"}

type State uint

type StateMachine struct {
	currentState  State
	previousState State
}

func Build() StateMachine {
	return StateMachine{S_START, S_START}
}

func (sm *StateMachine) State() State {
	return sm.currentState
}

func (sm *StateMachine) SetState(state State) {
	if sm.currentState == state {
		return
	}
	log.Debugf("Переход состояния %s -> %s", stateName(sm.currentState), stateName(state))
	switch sm.currentState {
	case S_START:
		if state != S_READY && state != S_FAIL {
			sm.illegalTransition(state)
		}
	case S_READY:
		if state != S_LINKS_LOAD && state != S_PING && state != S_ORI_CMD && state != S_START {
			sm.illegalTransition(state)
		}
	case S_LINKS_LOAD:
		if state != S_READY {
			sm.illegalTransition(state)
		}
	case S_PING:
	case S_ORI_CMD:
		if state != S_READY && state != S_ORI_CMD_SEND_VER && state != S_ORI_CMD_SEND_DATE &&
			state != S_ORI_CMD_SEND_TIME && state != S_ORI_CMD_CLIENT_REG && state != S_ORI_CMD_FILE &&
			state != S_ORI_CMD_CHK_SERVER_REG {
			sm.illegalTransition(state)
		}
	case S_ORI_CMD_SEND_VER:
	case S_ORI_CMD_SEND_DATE:
	case S_ORI_CMD_SEND_TIME:
	case S_ORI_CMD_CLIENT_REG:
		if state != S_READY {
			sm.illegalTransition(state)
		}
	case S_ORI_CMD_FILE:
		if state != S_READY {
			sm.illegalTransition(state)
		}
	case S_FAIL:
	default:
		log.Panicf("Недопустимое состояние: %d", sm.currentState)
	}
	sm.previousState = sm.currentState
	sm.currentState = state
}

//
// Возвращает текстовое название состояние машины
//
func stateName(state State) string {
	n, found := stateNames[uint(state)]
	if !found {
		n = "S_" + strconv.FormatUint(uint64(state), 10)
	}
	return n
}

func (sm *StateMachine) illegalTransition(newState State) {
	log.Panicf("Недопустимый переход из состояния %s в состояние %s", stateName(sm.currentState), stateName(newState))
}
