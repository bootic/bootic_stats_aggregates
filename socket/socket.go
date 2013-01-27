package socket

import(
  "regexp"
  "fmt"
  "bootic_stats_aggregates/data"
  "github.com/bitly/go-simplejson"
  zmq "github.com/alecthomas/gozmq"
)

type Daemon struct {
  socket zmq.Socket
  observers map[string][]data.EventsChannel
}

func (d *Daemon) listen() {
  for {
    msg, _ := d.socket.Recv(0)
    
    reg, _ := regexp.Compile(`^(?:\w+)?\s+(.+)`)
    r := reg.FindStringSubmatch(string(msg))
    
    payload := r[1]
    event, jsonErr := simplejson.NewJson([]byte(payload))
    
    if jsonErr != nil {
      panic(jsonErr)
    }
    
    d.Dispatch(event)
  }
}

func (self *Daemon) SubscribeToType(observer data.EventsChannel, typeStr string) {
  self.observers[typeStr] = append(self.observers[typeStr], observer)
}

func (self *Daemon) Dispatch(event *simplejson.Json) {
  // Dispatch to global observers
  for _, observer := range self.observers["all"] {
    observer <- event
  }
  
  // Dispatch to type observers
  evtStr, _ := event.Get("type").String()
  for _, observer := range self.observers[evtStr] {
    observer <- event
  }
}

func NewZMQSubscriber(host, topic string) (daemon *Daemon, err error) {
  context, _ := zmq.NewContext()
  socket, err := context.NewSocket(zmq.SUB)
  
  socket.SetSockOptString(zmq.SUBSCRIBE, topic)

  socket.Connect(host)
  fmt.Println("ZMQ client connected to", host)
  
  daemon = &Daemon{
    socket: socket,
    observers: make(map[string][]data.EventsChannel),
  }
  
  go daemon.listen()
  
  return
}