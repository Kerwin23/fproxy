package store

import (
	"errors"
	"github.com/garyburd/redigo/redis"
	"strconv"
	"time"
)

type RedisManager struct {
	redisPool *redis.Pool
}

func NewRedisManager(host string, port int, password string, db, maxIdle, maxActive int, timeout time.Duration) (*RedisManager, error) {
	if host == "" && port < 0 {
		return nil, errors.New("host and port can not be null")
	}
	dialAddr := host + ":" + strconv.Itoa(port)
	redisPool := &redis.Pool{
		MaxIdle:     maxIdle,
		MaxActive:   maxActive,
		IdleTimeout: timeout,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", dialAddr, redis.DialPassword(password), redis.DialDatabase(db))
			if err != nil {
				return nil, err
			}
			return c, nil
		},
	}
	return &RedisManager{redisPool: redisPool}, nil
}

func (r *RedisManager) getConn() redis.Conn {
	return r.redisPool.Get()
}

func (r *RedisManager) releaseConn(conn redis.Conn) {
	if conn == nil {
		return
	}
	conn.Close()
}

func (r *RedisManager) Set(key string, value string) {
	conn := r.getConn()
	defer r.releaseConn(conn)
	conn.Do("SET", key, value)
}

func (r *RedisManager) Get(key string) (string, error) {
	conn := r.getConn()
	defer r.releaseConn(conn)
	return redis.String(conn.Do("GET", key))
}

func (r *RedisManager) Sadd(key string, members ...string) (int, error) {
	conn := r.getConn()
	defer r.releaseConn(conn)
	return redis.Int(conn.Do("SADD", key, members))
}

func (r *RedisManager) Sismember(key string, member string) (bool, error) {
	conn := r.getConn()
	defer r.releaseConn(conn)
	return redis.Bool(conn.Do("SISMEMBER", key, member))
}

func (r *RedisManager) Smembers(key string) ([][]byte, error) {
	conn := r.getConn()
	defer r.releaseConn(conn)
	return redis.ByteSlices(conn.Do("SMEMBERS", key))
}

func (r *RedisManager) Lpop(key string) (string, error) {
	conn := r.getConn()
	defer r.releaseConn(conn)
	return redis.String(conn.Do("LPOP", key))
}

func (r *RedisManager) Rpush(key string, value string) {
	conn := r.getConn()
	defer r.releaseConn(conn)
	conn.Do("RPUSH", key, value)
}

func (r *RedisManager) Lrange(key string, start, stop int) ([][]byte, error) {
	conn := r.getConn()
	defer r.releaseConn(conn)
	return redis.ByteSlices(conn.Do("LRANGE", key, strconv.Itoa(start), strconv.Itoa(stop)))
}

func (r *RedisManager) Len(key string) (int, error) {
	conn := r.getConn()
	defer r.releaseConn(conn)
	return redis.Int(conn.Do("LEN", key))
}

func (r *RedisManager) RpopLpush(source, destination string) {
	conn := r.getConn()
	defer r.releaseConn(conn)
	conn.Do("RPOPLPUSH", source, destination)
}

func (r *RedisManager) Rename(key, newkey string) (int, error) {
	conn := r.getConn()
	defer r.releaseConn(conn)
	return redis.Int(conn.Do("RENAME", key, newkey))
}

func (r *RedisManager) Del(key string) {
	conn := r.getConn()
	defer r.releaseConn(conn)
	conn.Do("DEL", key)
}

func (r *RedisManager) Incr(key string) (int64, error) {
	conn := r.getConn()
	defer r.releaseConn(conn)
	return redis.Int64(conn.Do("INCR", key))
}
