package main

import(
  "log"
  "bootic_stats_aggregates/socket"
  "bootic_stats_aggregates/redis_stats"
  "bootic_stats_aggregates/handlers"
  "net/http"
  "github.com/gorilla/mux"
  "flag"
)

func main() {
  var(
    topic string
    zmqAddress string
    redisAddress string
    httpHost string
    pathprefix string
  )
  
  flag.StringVar(&topic, "topic", "", "ZMQ topic to subscribe to") // event type. ie "order", "pageview"
  flag.StringVar(&zmqAddress, "zmqsocket", "tcp://127.0.0.1:6000", "ZMQ socket address to bind to")
  flag.StringVar(&redisAddress, "redishost", "localhost:6379", "Redis host:port")
  flag.StringVar(&httpHost, "httphost", "localhost:8001", "HTTP host:port for JSON API")
  flag.StringVar(&pathprefix, "pathprefix", "/stats", "Path prefix for HTTP API")
  
  flag.Parse()
  
  // Setup ZMQ subscriber +++++++++++++++++++++++++++++++
  daemon, _  := zmq.NewZMQSubscriber(zmqAddress, topic)
  
  log.Println("ZMQ socket started on", zmqAddress, "topic '", topic, "'")
  
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
  
  rootHandler := handlers.RootHandler(tracker.Conn, pathprefix)
  keyHandler := handlers.KeyHandler(tracker.Conn, pathprefix)
  allKeysHandler := handlers.AllKeysHandler(tracker.Conn, pathprefix)
  
  router.HandleFunc("/", rootHandler).Methods("GET")
  router.HandleFunc("/favicon.ico", handlers.Favicon).Methods("GET")
  router.HandleFunc("/{chartType}", allKeysHandler).Methods("GET")
  router.HandleFunc("/{chartType}/{key}", allKeysHandler).Methods("GET")
  router.HandleFunc("/{chartType}/{key}/{evt}", allKeysHandler).Methods("GET")
  
  router.HandleFunc("/{chartType}/{key}/{evt}/{year}", keyHandler).Methods("GET")
  router.HandleFunc("/{chartType}/{key}/{evt}/{year}/{month}", keyHandler).Methods("GET")
  router.HandleFunc("/{chartType}/{key}/{evt}/{year}/{month}/{day}", keyHandler).Methods("GET")
  
  http.Handle("/", http.StripPrefix(pathprefix, router))
  
  // Start HTTP server
  log.Println("Starting HTTP server on", httpHost)
  log.Fatal(http.ListenAndServe(httpHost, nil))
}