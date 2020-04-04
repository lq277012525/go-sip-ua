package invite

import (
	"fmt"

	"github.com/ghettovoice/gosip/log"
	"github.com/ghettovoice/gosip/sip"
	"github.com/pixelbender/go-sdp/sdp"
)

var (
	logger log.Logger
)

func init() {
	logger = log.NewDefaultLogrusLogger().WithPrefix("invite.Session")
}

type Status string

const (
	Null           Status = "Null"
	InviteSent     Status = "InviteSent"     /**< After INVITE s sent */
	InviteReceived Status = "InviteReceived" /**< After INVITE s received. */
	//Offer          Status = "Offer"          /**< After re-INVITE/UPDATE s received */
	//Answer         Status = "Answer"         /**< After response for re-INVITE/UPDATE. */
	Provisional      Status = "Provisional" /**< After response for 1XX. */
	EarlyMedia       Status = "EarlyMedia"  /**< After response 1XX with sdp. */
	WaitingForAnswer Status = "WaitingForAnswer"
	WaitingForACK    Status = "WaitingForACK" /**< After 2xx s sent/received. */
	Answered         Status = "Answered"
	Canceled         Status = "Canceled"
	Confirmed        Status = "Confirmed"  /**< After ACK s sent/received. */
	Failure          Status = "Failure"    /**< Session s rejected or canceled. */
	Terminated       Status = "Terminated" /**< Session s terminated. */
)

type Direction string

const (
	Outgoing Direction = "Outgoing"
	Incoming Direction = "Incoming"
)

type InviteSessionHandler func(s *Session, req *sip.Request, resp *sip.Response, status Status)

type Session struct {
	contact     *sip.Address
	status      Status
	id          sip.CallID
	offer       *sdp.Session
	answer      *sdp.Session
	request     sip.Request
	response    sip.Response
	transaction sip.Transaction
	direction   Direction
}

func NewInviteSession(contact *sip.Address, req sip.Request, cid sip.CallID, tx sip.Transaction, dir Direction) *Session {
	s := &Session{
		request:     req,
		id:          cid,
		transaction: tx,
		direction:   dir,
		contact:     contact,
	}
	return s
}

func (s *Session) isInProgress() bool {
	switch s.status {
	case Null:
		fallthrough
	case InviteSent:
		fallthrough
	case Provisional:
		fallthrough
	case InviteReceived:
		fallthrough
	case WaitingForAnswer:
		return true
	default:
		return false
	}
}

func (s *Session) isEstablished() bool {
	switch s.status {
	case Answered:
		fallthrough
	case WaitingForACK:
		fallthrough
	case Confirmed:
		return true
	default:
		return false
	}
}

func (s *Session) isEnded() bool {
	switch s.status {
	case Canceled:
		fallthrough
	case Terminated:
		return true
	default:
		return false
	}
}

func (s *Session) StoreRequest(request sip.Request) {
	s.request = request
}

func (s *Session) StoreResponse(response sip.Response) {
	s.response = response
}

func (s *Session) StoreTransaction(tx sip.Transaction) {
	s.transaction = tx
}

func (s *Session) SetState(status Status) {
	s.status = status
}

func (s *Session) Status() Status {
	return s.status
}

func (s *Session) Direction() Direction {
	return s.direction
}

// GetEarlyMedia Get sdp for early media.
func (s *Session) GetEarlyMedia() *sdp.Session {
	return s.answer
}

//ProvideOffer .
func (s *Session) ProvideOffer(sdp *sdp.Session) {
	s.offer = sdp
}

// ProvideAnswer .
func (s *Session) ProvideAnswer(sdp *sdp.Session) {
	s.answer = sdp
}

//Info Send Info
func (s *Session) Info(content *sip.String) {

}

// Reject Reject incoming call or for re-INVITE or UPDATE,
func (s *Session) Reject(statusCode sip.StatusCode, reason string) {
	tx := (s.transaction.(sip.ServerTransaction))
	request := s.request
	logger.Infof("Reject: Request => %s, body => %s", request.Short(), request.Body())
	response := sip.NewResponseFromRequest(request.MessageID(), request, statusCode, reason, "")
	tx.Respond(response)
}

//End end session
func (s *Session) End(statusCode sip.StatusCode, reason string) error {

	if s.status == Terminated {
		return fmt.Errorf("Invalid status: %v", s.status)
	}

	switch s.status {
	// - UAC -
	case Null:
		fallthrough
	case InviteSent:
		fallthrough
	case Provisional:
		fallthrough
	case EarlyMedia:
		logger.Info("Canceling session.")

	// - UAS -
	case WaitingForAnswer:
		fallthrough
	case Answered:
		logger.Info("Rejecting session")

	case WaitingForACK:
		fallthrough
	case Confirmed:
		logger.Info("Terminating session.")

	}

	return nil
}

// Accept 200
func (s *Session) Accept(statusCode sip.StatusCode) {
	tx := (s.transaction.(sip.ServerTransaction))

	if s.answer == nil {
		logger.Errorf("Answer sdp is nil!")
		return
	}
	request := s.request
	response := sip.NewResponseFromRequest(request.MessageID(), request, 200, "OK", s.answer.String())

	hdrs := request.GetHeaders("Content-Type")
	if len(hdrs) == 0 {
		contentType := sip.ContentType("application/sdp")
		response.AppendHeader(&contentType)
	} else {
		sip.CopyHeaders("Content-Type", request, response)
	}
	/*
		to, _ := request.To()
		contact := sip.ContactHeader{Address: to.Address}
		util.BuildContactHeader("Contact", request, response, nil)
	*/
	response.AppendHeader(s.contact.AsContactHeader())
	s.response = response
	tx.Respond(response)
}

// Redirect send a 3xx
func (s *Session) Redirect(addr *sip.Address, code sip.StatusCode) {

}

// Provisional send a provisional code 100|180|183
func (s *Session) Provisional(statusCode sip.StatusCode, reason string) {
	tx := (s.transaction.(sip.ServerTransaction))
	request := s.request
	var response sip.Response
	if s.answer != nil {
		sdp := s.answer.String()
		response = sip.NewResponseFromRequest(request.MessageID(), request, statusCode, reason, sdp)
		hdrs := response.GetHeaders("Content-Type")
		if len(hdrs) == 0 {
			contentType := sip.ContentType("application/sdp")
			response.AppendHeader(&contentType)
		} else {
			sip.CopyHeaders("Content-Type", request, response)
		}
	} else {
		response = sip.NewResponseFromRequest(request.MessageID(), request, statusCode, reason, "")
	}

	s.response = response
	tx.Respond(response)
}
