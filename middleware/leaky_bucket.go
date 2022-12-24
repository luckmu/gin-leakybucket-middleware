package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

var lua_script = `
-- KEYS = {RL:ip:::1}
-- ARGV = {freq, expiration_secs, request_id}

local key = KEYS[1]
local lock = key .. ':lock'
local freq, expiration_secs, request_id = ARGV[1], ARGV[2], ARGV[3]

local r = redis.call('SET', lock, request_id, 'NX', 'EX', 1)
if not r then
    return r
end

local ret = 0
local cnt = redis.call('GET', key)
cnt = tonumber(cnt)

if cnt then
    if cnt > 0 then
        redis.call('DECR', key)
    else
        ret = redis.call('PTTL', key)
    end
else
    redis.call('SET', key, freq-1, 'NX', 'EX', expiration_secs)
end

r = redis.call('GET', lock)
if r == request_id then
    redis.call('DEL', lock)
end

return ret
`

var (
	redis_cli   *redis.Client
	script_sha1 string
)

func init() {
	redis_cli = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	sha1, err := redis_cli.ScriptLoad(lua_script).Result()
	if err != nil {
		panic(err)
	}
	script_sha1 = sha1
}

// init_timers:
// must call `defer stop()`
func init_timers(timeout time.Duration) (wait1 func() <-chan time.Time, wait2 func(duration time.Duration) bool, stop func()) {
	timer, waiter := time.NewTimer(timeout), time.NewTimer(-1)
	// fire waiter
	<-waiter.C
	wait1 = func() <-chan time.Time {
		return timer.C
	}
	wait2 = func(duration time.Duration) bool {
		waiter.Reset(duration)
		select {
		case <-timer.C:
			return false
		case <-waiter.C:
			return true
		}
	}
	stop = func() {
		timer.Stop()
		waiter.Stop()
	}
	return
}

// RLimiter
func RLimiter(key string, freq, duration int) gin.HandlerFunc {
	if freq <= 0 {
		panic("`freq` must be greater than 0")
	}
	limit_prefix := "RL:" + key + ":"

	return func(c *gin.Context) {
		limit_obj, request_id := c.GetString(key), c.GetString("Request-ID")
		wait1, wait2, stop := init_timers(time.Second * 3)
		defer stop()

		for retries := 2; retries >= 0; retries-- {
			select {
			case <-wait1():
				c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{"msg": "request timeout"})
				return
			default:
				r, err := redis_cli.EvalSha(script_sha1, []string{limit_prefix + limit_obj}, freq, duration, request_id).Result()
				if err != nil && err != redis.Nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
					return
				}
				wait_refresh, ok := r.(int64)
				if !ok {
					// fmt.Println("cannot get lock, retrying...")
					if !wait2(time.Millisecond * 100) {
						c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{"msg": "request timeout"})
						return
					}
				} else if wait_refresh > 0 {
					// fmt.Printf("no tokens, wait %v s\n", wait_refresh/1000)
					if !wait2(time.Millisecond * time.Duration(wait_refresh)) {
						c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{"msg": "request timeout"})
						return
					}
				} else {
					return
				}
			}
		}
		c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{"msg": "exceed max retries"})
	}
}
