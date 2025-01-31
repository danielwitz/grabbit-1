package saga

import (
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/wework/grabbit/gbus"
)

var _ gbus.HandlerRegister = &Def{}

type MsgToFuncPair struct {
	Filter       *gbus.MessageFilter
	SagaFuncName string
}

//Def defines a saga type
type Def struct {
	glue        *Glue
	sagaType    reflect.Type
	startedBy   []string
	lock        *sync.Mutex
	sagaConfFns []gbus.SagaConfFn
	msgToFunc   []*MsgToFuncPair
}

//HandleMessage implements HandlerRegister interface
func (sd *Def) HandleMessage(message gbus.Message, handler gbus.MessageHandler) error {
	sd.addMsgToHandlerMapping("", sd.glue.svcName, message, handler)
	return sd.glue.registerMessage(message)
}

//HandleEvent implements HandlerRegister interface
func (sd *Def) HandleEvent(exchange, topic string, event gbus.Message, handler gbus.MessageHandler) error {
	sd.addMsgToHandlerMapping(exchange, topic, event, handler)
	return sd.glue.registerEvent(exchange, topic, event)
}

func (sd *Def) getHandledMessages() []string {
	messages := make([]string, 0)
	for _, pair := range sd.msgToFunc {
		if pair.Filter.MsgName != "" {
			messages = append(messages, pair.Filter.MsgName)
		}
	}
	return messages
}

func (sd *Def) addMsgToHandlerMapping(exchange, routingKey string, message gbus.Message, handler gbus.MessageHandler) {

	fn := getFunNameFromHandler(handler)

	msgToFunc := &MsgToFuncPair{
		Filter:       gbus.NewMessageFilter(exchange, routingKey, message),
		SagaFuncName: fn}
	sd.msgToFunc = append(sd.msgToFunc, msgToFunc)
}

func getFunNameFromHandler(handler gbus.MessageHandler) string {
	funName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
	splits := strings.Split(funName, ".")
	fn := strings.Replace(splits[len(splits)-1], "-fm", "", -1)
	return fn
}

func (sd *Def) newInstance() *Instance {
	instance := NewInstance(sd.sagaType, sd.msgToFunc)
	return sd.configureSaga(instance)
}

func (sd *Def) configureSaga(instance *Instance) *Instance {
	saga := instance.UnderlyingInstance
	for _, conf := range sd.sagaConfFns {
		saga = conf(saga)
	}
	return instance
}

func (sd *Def) shouldStartNewSaga(message *gbus.BusMessage) bool {
	fqn := message.PayloadFQN

	for _, starts := range sd.startedBy {
		if fqn == starts {
			return true
		}
	}
	return false
}

// func (sd *Def) canTimeout() bool {
// 	sd.sagaType.Implements(u reflect.Type)
// }

func (sd *Def) String() string {
	return gbus.GetTypeFQN(sd.sagaType)
}
