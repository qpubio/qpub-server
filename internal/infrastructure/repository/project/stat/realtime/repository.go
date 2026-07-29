package realtime

import (
	"github.com/qpubio/qpub-server/internal/domain/project/stat/realtime"

	"github.com/go-redis/redis"
)

type repository struct {
	redis *redis.Client
}

func NewRepository(redis *redis.Client) realtime.Repository {
	return &repository{redis: redis}
}

func (r *repository) Incr(key realtime.Key) error {
	return r.redis.Incr(key.String()).Err()
}

func (r *repository) IncrBy(key realtime.Key, value int64) error {
	return r.redis.IncrBy(key.String(), value).Err()
}

func (r *repository) Decr(key realtime.Key) error {
	script := `
		local current = redis.call('get', KEYS[1])
		if current and tonumber(current) > 0 then
			return redis.call('decr', KEYS[1])
		end
		return redis.call('get', KEYS[1]) or "0"
	`
	return r.redis.Eval(script, []string{key.String()}).Err()
}

func (r *repository) DecrBy(key realtime.Key, value int64) error {
	script := `
		local current = redis.call('get', KEYS[1])
		if current then
			local newValue = tonumber(current) - tonumber(ARGV[1])
			if newValue >= 0 then
				return redis.call('decrby', KEYS[1], ARGV[1])
			else
				redis.call('set', KEYS[1], 0)
				return 0
			end
		end
		redis.call('set', KEYS[1], 0)
		return 0
	`
	return r.redis.Eval(script, []string{key.String()}, value).Err()
}

func (r *repository) Get(key realtime.Key) (int64, error) {
	return r.redis.Get(key.String()).Int64()
}

func (r *repository) Set(key realtime.Key, value int64) error {
	return r.redis.Set(key.String(), value, 0).Err()
}

func (r *repository) GetByPattern(pattern string) ([]string, error) {
	keys, err := r.redis.Keys(pattern).Result()
	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (r *repository) Reset(key realtime.Key) error {
	return r.redis.Del(key.String()).Err()
}

func (r *repository) ResetByPattern(pattern string) error {
	script := `
        local keys = redis.call('KEYS', ARGV[1])
        if #keys > 0 then
            return redis.call('DEL', unpack(keys))
        end
        return 0
    `
	return r.redis.Eval(script, []string{}, pattern).Err()
}

// BatchIncr performs multiple increment operations in a pipeline
func (r *repository) BatchIncr(keys []realtime.Key) error {
	pipe := r.redis.Pipeline()
	for _, key := range keys {
		pipe.Incr(key.String())
	}
	_, err := pipe.Exec()
	return err
}

// BatchIncrBy performs multiple increment by value operations in a pipeline
func (r *repository) BatchIncrBy(operations map[realtime.Key]int64) error {
	pipe := r.redis.Pipeline()
	for key, value := range operations {
		pipe.IncrBy(key.String(), value)
	}
	_, err := pipe.Exec()
	return err
}

// BatchDecr performs multiple safe decrement operations
func (r *repository) BatchDecr(keys []realtime.Key) error {
	script := `
		for i, key in ipairs(KEYS) do
			local current = redis.call('get', key)
			if current and tonumber(current) > 0 then
				redis.call('decr', key)
			end
		end
		return "OK"
	`

	keyStrings := make([]string, len(keys))
	for i, key := range keys {
		keyStrings[i] = key.String()
	}

	return r.redis.Eval(script, keyStrings).Err()
}

// BatchDecrBy performs multiple safe decrement by value operations
func (r *repository) BatchDecrBy(operations map[realtime.Key]int64) error {
	script := `
		local keyIndex = 1
		for i = 1, #ARGV, 2 do
			local key = KEYS[keyIndex]
			local value = tonumber(ARGV[i])
			local current = redis.call('get', key)
			if current then
				local newValue = tonumber(current) - value
				if newValue >= 0 then
					redis.call('decrby', key, value)
				else
					redis.call('set', key, 0)
				end
			else
				redis.call('set', key, 0)
			end
			keyIndex = keyIndex + 1
		end
		return "OK"
	`

	var keys []string
	var args []interface{}

	for key, value := range operations {
		keys = append(keys, key.String())
		args = append(args, value)
	}

	return r.redis.Eval(script, keys, args...).Err()
}

// BatchReset performs multiple reset operations in a pipeline
func (r *repository) BatchReset(keys []realtime.Key) error {
	pipe := r.redis.Pipeline()
	for _, key := range keys {
		pipe.Del(key.String())
	}
	_, err := pipe.Exec()
	return err
}
