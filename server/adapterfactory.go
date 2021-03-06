package server

import (
  "net"
  "strconv"

  "github.com/go-redis/redis/v7"
)

type AdapterFactory struct {
  pubClient *redis.Client
  subClient *redis.Client

  NewAdapter func(room string) Adapter
}

func NewAdapterFactory(
  loggerFactory LoggerFactory,
  c StoreConfig,
) *AdapterFactory {
  var log = loggerFactory.GetLogger("adapterfactory")

  f := AdapterFactory{}

  switch c.Type {
  case StoreTypeRedis:
    addr := net.JoinHostPort(c.Redis.Host, strconv.Itoa(c.Redis.Port))
    prefix := c.Redis.Prefix
    log.Printf("Using RedisAdapter: %s with prefix %s", addr, prefix)
    opts := &redis.Options{
      Addr: addr,
      DB:   c.Redis.DB,
    }
    f.pubClient = redis.NewClient(opts)
    f.subClient = redis.NewClient(opts)
    f.NewAdapter = func(room string) Adapter {
      return NewRedisAdapter(loggerFactory, f.pubClient, f.subClient, prefix, room)
    }
  case StoreTypeRedisSentinel:
    prefix := c.Redis.Prefix
    log.Printf("Using Sentinel RedisAdapter: %v with prefix %s and master name %s", c.Sentinel.Addrs, prefix, c.Sentinel.Name)
    opts := &redis.FailoverOptions{
      SentinelAddrs: c.Sentinel.Addrs,
      MasterName:    c.Sentinel.Name,
      DB:            c.Sentinel.DB,
    }
    f.pubClient = redis.NewFailoverClient(opts)
    f.subClient = redis.NewFailoverClient(opts)
    f.NewAdapter = func(room string) Adapter {
      return NewRedisAdapter(loggerFactory, f.pubClient, f.subClient, prefix, room)
    }
  default:
    log.Printf("Using MemoryAdapter")
    f.NewAdapter = func(room string) Adapter {
      return NewMemoryAdapter(room)
    }
  }

  return &f
}

func (a *AdapterFactory) Close() (err error) {
  var errs []error
  if a.pubClient != nil {
    errs = append(errs, a.pubClient.Close())
  }
  if a.subClient != nil {
    errs = append(errs, a.subClient.Close())
  }
  return firstError(errs...)
}
