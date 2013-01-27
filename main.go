package main

import(
  "log"
  "bootic_stats_aggregates/socket"
  "bootic_stats_aggregates/redis_stats"
  "bootic_stats_aggregates/handlers"
  "net/http"
  "github.com/gorilla/mux"
)

func main() {
  
  topic         := "" // event type. ie "order", "pageview"
  zmqAddress    := "tcp://127.0.0.1:6000"
  redisAddress  := "localhost:6379"
  httpHost      := "localhost:8001"
  
  // Setup ZMQ subscriber +++++++++++++++++++++++++++++++
  daemon, _  := socket.NewZMQSubscriber(zmqAddress, topic)
  
  log.Println("ZMQ socket started on", zmqAddress)
  
  // Setup Rediss trackr ++++++++++++++++++++++++++++++++
  tracker, err := redis_stats.NewTracker(redisAddress)
  
  log.Println("Redis tracker started on", redisAddress)
  
  if err != nil {
    panic(err)
  }
  
  // Redis subscribe to these events
  daemon.SubscribeToType(tracker.Notifier, "pageview")
  daemon.SubscribeToType(tracker.Funnels, "order")
  
  log.Println("Redis tracking 'pageview' and 'order' events")
  
  // Declare HTTP API routes ++++++++++++++++++++++++++++
  router := mux.NewRouter()
  
  rootHandler := handlers.RootHandler(tracker.Conn)
  keyHandler := handlers.KeyHandler(tracker.Conn)
  allKeysHandler := handlers.AllKeysHandler(tracker.Conn)
  
  router.HandleFunc("/", rootHandler).Methods("GET")
  router.HandleFunc("/{chartType}", allKeysHandler).Methods("GET")
  router.HandleFunc("/{chartType}/{key}", allKeysHandler).Methods("GET")
  router.HandleFunc("/{chartType}/{key}/{evt}", allKeysHandler).Methods("GET")
  
  router.HandleFunc("/{chartType}/{key}/{evt}/{year}", keyHandler).Methods("GET")
  router.HandleFunc("/{chartType}/{key}/{evt}/{year}/{month}", keyHandler).Methods("GET")
  router.HandleFunc("/{chartType}/{key}/{evt}/{year}/{month}/{day}", keyHandler).Methods("GET")
  
  http.Handle("/", router)
  
  // Start HTTP server
  log.Println("Starting HTTP server on", httpHost)
  log.Fatal(http.ListenAndServe(httpHost, nil))
}