package proxy

import (
	"log"
	"regexp"
	"strings"
	//"errors"

	"github.com/etclabscore/open-etc-pool/rpc"
	"github.com/etclabscore/open-etc-pool/util"
)

// Allow only lowercase hexadecimal with 0x prefix
var noncePattern = regexp.MustCompile("^0x[0-9a-f]{16}$")
var hashPattern = regexp.MustCompile("^0x[0-9a-f]{64}$")
var workerPattern = regexp.MustCompile("^[0-9a-zA-Z-_]{1,32}$")

// Stratum
func (s *ProxyServer) handleLoginRPC(cs *Session, params []string, id string) (bool, *ErrorReply) {
	var loginCheck string
	if len(params) == 0 {
		return false, &ErrorReply{Code: -1, Message: "Invalid params"}
	}

	login := strings.ToLower(params[0])
	if strings.Contains(login, ".") {
		longString := strings.Split(login, ".")
                loginCheck = longString[0]
    } else {
        loginCheck = login
    }

	if !util.IsValidHexAddress(loginCheck) {
		return false, &ErrorReply{Code: -1, Message: "Invalid login"}
	}
	if !s.policy.ApplyLoginPolicy(login, cs.ip) {
		return false, &ErrorReply{Code: -1, Message: "You are blacklisted"}
	}
	cs.login = login
	s.registerSession(cs)
	log.Printf("Stratum miner connected %v@%v", login, cs.ip)
	return true, nil
}

func (s *ProxyServer) handleGetWorkRPC(cs *Session) ([]string, *ErrorReply) {
	t := s.currentBlockTemplate()
	if t == nil || len(t.Header) == 0 || s.isSick() {
		return nil, &ErrorReply{Code: 0, Message: "Work not ready"}
	}
		return []string{t.Header, t.Seed, s.diff, util.ToHex(int64(t.Height))}, nil
}

// Stratum
func (s *ProxyServer) handleTCPSubmitRPC(cs *Session, id string, params []string) (bool, *ErrorReply) {
	s.sessionsMu.RLock()
	_, ok := s.sessions[cs]
	s.sessionsMu.RUnlock()

	if !ok {
		return false, &ErrorReply{Code: 25, Message: "Not subscribed"}
	}
	return s.handleSubmitRPC(cs, cs.login, id, params)
}

func (s *ProxyServer) handleSubmitRPC(cs *Session, login, id string, params []string) (bool, *ErrorReply) {
	if strings.Contains(login, ".") {
		longString := strings.Split(login, ".")
		id = longString[1]
		login = longString[0]
    }

	if !workerPattern.MatchString(id){
		id = "default"
	}
	if len(params) != 3 {
		s.policy.ApplyMalformedPolicy(cs.ip)
		log.Printf("Malformed params from %s@%s %v", login, cs.ip, params)
		return false, &ErrorReply{Code: -1, Message: "Invalid params"}
	}

    stratumMode := cs.stratumMode()
	if stratumMode != EthProxy {
		for i := 0; i <= 2; i++ {
			if params[i][0:2] != "0x" {
				params[i] = "0x" + params[i]
			}
		}
	}

	if !noncePattern.MatchString(params[0]) || !hashPattern.MatchString(params[1]) || !hashPattern.MatchString(params[2]) {
		s.policy.ApplyMalformedPolicy(cs.ip)
		log.Printf("Malformed PoW result from %s@%s %v", login, cs.ip, params)
		return false, &ErrorReply{Code: -1, Message: "Malformed PoW result"}
	}

	//go func(s *ProxyServer, cs *Session, login, id string, params []string) {
		t := s.currentBlockTemplate()

		//MFO: 	This function (s.processShare) will process a share as per hasher.Verify function of github.com/ethereum/ethash
		//	output of this function is either:
		//		true,true   	(Exists) which means share already exists and it is validShare
		//		true,false		(Exists & invalid)which means share already exists and it is invalidShare or it is a block <-- should not ever happen
		//		false,false		(stale/invalid)which means share is new, and it is not a block, might be a stale share or invalidShare
		//		false,true		(valid)which means share is new, and it is a block or accepted share
		//	When this function finishes, the results is already recorded in the db for valid shares or blocks.
		exist, validShare := s.processShare(login, id, cs.ip, t, params, stratumMode != EthProxy)
		ok := s.policy.ApplySharePolicy(cs.ip, !exist && validShare)


		// if true,true or true,false
		if exist {
			log.Printf("Duplicate share from %s@%s %v", login, cs.ip, params)
			//cs.lastErr = errors.New("Duplicate share")
			return false, &ErrorReply{Code: 23, Message: "Invalid share"}
		}

		// if false, false
		if !validShare {
			//MFO: Here we have an invalid share
			log.Printf("Invalid share from %s@%s", login, cs.ip, params)
			// Bad shares limit reached, return error and close
			if !ok {
				return false, &ErrorReply{Code: 23, Message: "Invalid share"}
				//cs.lastErr = errors.New("Invalid share")
			}
			return false, nil
			//return false, &ErrorReply{Code: -1, Message: "Invalid share"}
		}

		if s.config.Proxy.Debug {
			//MFO: Here we have a valid share and it is already recorded in DB by miner.go
			// if false, true
			log.Printf("Valid share from %s@%s", login, cs.ip)
		}

		if !ok {
			log.Printf("High rate of invalid shares from %s@%s", login, cs.ip)
			//cs.lastErr = errors.New("High rate of invalid shares")
		}
	//}(s, cs, login, id, params)
	//log.Printf("TEST", cs.lastErr)
	return true, nil
}

func (s *ProxyServer) handleGetBlockByNumberRPC() *rpc.GetBlockReplyPart {
	t := s.currentBlockTemplate()
	var reply *rpc.GetBlockReplyPart
	if t != nil {
		reply = t.GetPendingBlockCache
	}
	return reply
}

func (s *ProxyServer) handleUnknownRPC(cs *Session, m string) *ErrorReply {
	log.Printf("Unknown request method %s from %s", m, cs.ip)
	s.policy.ApplyMalformedPolicy(cs.ip)
	return &ErrorReply{Code: -3, Message: "Method not found"}
}
