package main

import(
  "log"
  "time"
  "bootic_stats_observer/socket"
  "bootic_stats_observer/redis_stats"
)

func main() {
  
	topic := "all"
	
  daemon, _  := socket.NewZMQSubscriber("tcp://127.0.0.1:6000", topic)
  
  log.Println("ZMQ socket started")
  
  tracker, err := redis_stats.NewTracker("localhost:6379")
  
  log.Println("Redis tracker started")
  
  if err != nil {
    panic(err)
  }
  
  daemon.SubscribeToType(tracker.Notifier, "pageview")
  daemon.SubscribeToType(tracker.Funnels, "order")
  
  log.Println("Redis tracking 'pageview' and 'order' events")
  
  time.Sleep(10e9)
}